package queue_jobs

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

var monthAbbr = [...]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

type catKey struct{ name, classification string }

type xlsxStyles struct {
	SectionTitle int
	ColHeader    int
	DataCell     int
	LabelCell    int
}

type GenerateCategoryReportJob struct {
	logger        *zap.Logger
	analyticsRepo repositories.AnalyticsRepositoryInterface
	ReportID      int64
	UserID        int64
	Params        models.CategoryReportParams
}

func (j *GenerateCategoryReportJob) Type() string { return TypeGenerateCategoryReport }

func NewGenerateCategoryReportJob(
	logger *zap.Logger,
	analyticsRepo repositories.AnalyticsRepositoryInterface,
	reportID, userID int64,
	params models.CategoryReportParams,
) *GenerateCategoryReportJob {
	return &GenerateCategoryReportJob{
		logger:        logger,
		analyticsRepo: analyticsRepo,
		ReportID:      reportID,
		UserID:        userID,
		Params:        params,
	}
}

func (j *GenerateCategoryReportJob) Process(ctx context.Context) error {
	if err := j.analyticsRepo.UpdateReport(ctx, nil, j.ReportID, map[string]interface{}{
		"status": "processing",
	}); err != nil {
		return err
	}

	rows, err := j.analyticsRepo.FetchCategoryReportData(
		ctx, nil,
		j.UserID,
		j.Params.InflowCategoryIDs,
		j.Params.OutflowCategoryIDs,
		j.Params.Years,
		j.Params.AllTime,
		j.Params.Description,
	)
	if err != nil {
		return j.fail(ctx, err)
	}

	categoryLabel := deriveCategoryLabel(rows)

	data, err := j.buildXLSX(rows)
	if err != nil {
		return j.fail(ctx, err)
	}

	filePath, err := j.saveFile(data)
	if err != nil {
		return j.fail(ctx, err)
	}

	now := time.Now().UTC()
	fileSize := int64(len(data))
	return j.analyticsRepo.UpdateReport(ctx, nil, j.ReportID, map[string]interface{}{
		"status":       "completed",
		"name":         j.reportName(categoryLabel),
		"file_path":    filePath,
		"file_size":    fileSize,
		"completed_at": now,
	})
}

func (j *GenerateCategoryReportJob) fail(ctx context.Context, err error) error {
	j.logger.Error("category report generation failed", zap.Int64("reportID", j.ReportID), zap.Error(err))
	msg := err.Error()
	_ = j.analyticsRepo.UpdateReport(ctx, nil, j.ReportID, map[string]interface{}{
		"status": "failed",
		"error":  msg,
	})
	return err
}

func (j *GenerateCategoryReportJob) buildXLSX(rows []models.CategoryReportDataRow) ([]byte, error) {
	byYear := make(map[int][]models.CategoryReportDataRow)
	for _, r := range rows {
		byYear[r.Year] = append(byYear[r.Year], r)
	}
	years := sortedYears(byYear)

	f := excelize.NewFile()
	defer func(f *excelize.File) {
		err := f.Close()
		if err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}(f)

	styles := j.makeStyles(f)

	err := f.SetSheetName("Sheet1", "Summary")
	if err != nil {
		return nil, err
	}
	j.writeSummarySheet(f, "Summary", styles, rows, years)

	if j.Params.AllTime && len(years) > 1 {
		if _, err := f.NewSheet("All Time"); err != nil {
			return nil, err
		}
		j.writeAllTimeSheet(f, "All Time", styles, rows, years)
	}

	for _, year := range years {
		name := fmt.Sprintf("%d", year)
		if _, err := f.NewSheet(name); err != nil {
			return nil, err
		}
		j.writeYearSheet(f, name, styles, year, byYear[year])
	}

	summaryIdx, _ := f.GetSheetIndex("Summary")
	f.SetActiveSheet(summaryIdx)

	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (j *GenerateCategoryReportJob) makeStyles(f *excelize.File) xlsxStyles {
	thin := []excelize.Border{
		{Type: "left", Color: "BFBFBF", Style: 1},
		{Type: "right", Color: "BFBFBF", Style: 1},
		{Type: "top", Color: "BFBFBF", Style: 1},
		{Type: "bottom", Color: "BFBFBF", Style: 1},
	}

	titleID, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
	})
	headerID, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"D6DCE4"}, Pattern: 1},
		Border:    thin,
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	dataID, _ := f.NewStyle(&excelize.Style{
		Border:    thin,
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	labelID, _ := f.NewStyle(&excelize.Style{
		Border:    thin,
		Alignment: &excelize.Alignment{Horizontal: "left"},
	})

	return xlsxStyles{SectionTitle: titleID, ColHeader: headerID, DataCell: dataID, LabelCell: labelID}
}

func (j *GenerateCategoryReportJob) writeSummarySheet(f *excelize.File, sheet string, styles xlsxStyles, rows []models.CategoryReportDataRow, years []int) {
	type yearTotals struct {
		primary   decimal.Decimal
		secondary decimal.Decimal
		months    map[int]struct{}
	}
	byYear := make(map[int]*yearTotals)
	for _, r := range rows {
		yt, ok := byYear[r.Year]
		if !ok {
			yt = &yearTotals{months: make(map[int]struct{})}
			byYear[r.Year] = yt
		}
		yt.months[r.Month] = struct{}{}
		if r.Classification == "inflow" {
			yt.primary = yt.primary.Add(r.Total)
		} else {
			yt.secondary = yt.secondary.Add(r.Total)
		}
	}

	cur := 1
	cur = xlsxTitle(f, sheet, cur, "Report Summary", styles.SectionTitle)
	cur++

	cur = xlsxHeaderRow(f, sheet, cur, []string{"Year", "Total Primary", "Total Secondary", "Effective", "Avg/Active Month (Eff)", "Avg/Calendar Month (Eff)", "Active Months"}, styles)

	effectives := make([]decimal.Decimal, 0, len(years))
	for _, year := range years {
		yt := byYear[year]
		eff := yt.primary.Sub(yt.secondary)
		effectives = append(effectives, eff)
		activeMonths := len(yt.months)
		activeAvg := decimal.Zero
		if activeMonths > 0 {
			activeAvg = eff.Div(decimal.NewFromInt(int64(activeMonths)))
		}
		calAvg := eff.Div(decimal.NewFromInt(int64(utils.CalendarMonths(year))))
		cur = xlsxDataRow(f, sheet, cur, []string{
			fmt.Sprintf("%d", year),
			yt.primary.StringFixed(2),
			yt.secondary.StringFixed(2),
			eff.StringFixed(2),
			activeAvg.StringFixed(2),
			calAvg.StringFixed(2),
			fmt.Sprintf("%d", activeMonths),
		}, 1, styles)
	}

	if len(years) > 1 {
		cur++
		cur = xlsxTitle(f, sheet, cur, "Year-over-Year Change", styles.SectionTitle)
		cur++

		cur = xlsxHeaderRow(f, sheet, cur, append([]string{"Metric"}, utils.YearStrings(years[1:])...), styles)

		yoyRow := []string{"YoY Change (Effective)"}
		yoyPctRow := []string{"YoY % Change"}
		for i := 1; i < len(years); i++ {
			diff := effectives[i].Sub(effectives[i-1])
			yoyRow = append(yoyRow, utils.SignedFixed(diff))
			if !effectives[i-1].IsZero() {
				pct := diff.Div(effectives[i-1].Abs()).Mul(decimal.NewFromInt(100))
				yoyPctRow = append(yoyPctRow, utils.SignedFixed(pct)+"%")
			} else {
				yoyPctRow = append(yoyPctRow, "-")
			}
		}
		cur = xlsxDataRow(f, sheet, cur, yoyRow, 1, styles)
		cur = xlsxDataRow(f, sheet, cur, yoyPctRow, 1, styles)

		slope := utils.LinearTrend(effectives)
		cur++
		cur = xlsxHeaderRow(f, sheet, cur, []string{"Annual Trend", "Value"}, styles)
		_ = xlsxDataRow(f, sheet, cur, []string{"Effective change per year", fmt.Sprintf("%s (%s)", utils.SignedFixed(slope), utils.TrendDirection(slope))}, 1, styles)
	}

	if err := f.SetColWidth(sheet, "A", "A", 24); err != nil {
		return
	}
	if err := f.SetColWidth(sheet, "B", "H", 22); err != nil {
		return
	}
}

func (j *GenerateCategoryReportJob) writeYearSheet(f *excelize.File, sheet string, styles xlsxStyles, year int, rows []models.CategoryReportDataRow) {
	monthSet := make(map[int]struct{})
	catMonthly := make(map[catKey]map[int]decimal.Decimal)
	catMonthlyCount := make(map[catKey]map[int]int)
	catDisplayClass := make(map[catKey]string)
	for _, r := range rows {
		monthSet[r.Month] = struct{}{}
		k := catKey{r.CategoryName, r.Classification}
		if catMonthly[k] == nil {
			catMonthly[k] = make(map[int]decimal.Decimal)
			catMonthlyCount[k] = make(map[int]int)
		}
		catMonthly[k][r.Month] = catMonthly[k][r.Month].Add(r.Total)
		catMonthlyCount[k][r.Month] += r.TxnCount
		catDisplayClass[k] = r.CategoryClassification
	}
	months := utils.SortedInts(monthSet)

	var inflows, outflows []catKey
	for k := range catMonthly {
		if k.classification == "inflow" {
			inflows = append(inflows, k)
		} else {
			outflows = append(outflows, k)
		}
	}
	sort.Slice(inflows, func(i, j int) bool { return inflows[i].name < inflows[j].name })
	sort.Slice(outflows, func(i, j int) bool { return outflows[i].name < outflows[j].name })

	monthCols := make([]string, len(months))
	for i, m := range months {
		monthCols[i] = monthAbbr[m-1]
	}

	calM := utils.CalendarMonths(year)

	buildCatRow := func(k catKey) []string {
		vals := []string{k.name, catDisplayClass[k]}
		var total decimal.Decimal
		activeM := 0
		for _, m := range months {
			v := catMonthly[k][m]
			vals = append(vals, v.StringFixed(2))
			total = total.Add(v)
			if !v.IsZero() {
				activeM++
			}
		}
		activeAvg := decimal.Zero
		if activeM > 0 {
			activeAvg = total.Div(decimal.NewFromInt(int64(activeM)))
		}
		calAvg := total.Div(decimal.NewFromInt(int64(calM)))
		return append(vals, total.StringFixed(2), activeAvg.StringFixed(2), calAvg.StringFixed(2))
	}

	catHeaders := append(append([]string{"Category", "Classification"}, monthCols...), "Total", "Avg/Active Month", "Avg/Calendar Month")

	cur := 1

	// --- Category Breakdown ---
	cur = xlsxTitle(f, sheet, cur, fmt.Sprintf("Category Breakdown - %d", year), styles.SectionTitle)
	cur++
	cur = xlsxHeaderRow(f, sheet, cur, catHeaders, styles)
	for _, k := range inflows {
		cur = xlsxDataRow(f, sheet, cur, buildCatRow(k), 2, styles)
	}
	if len(inflows) > 0 && len(outflows) > 0 {
		cur++
	}
	for _, k := range outflows {
		cur = xlsxDataRow(f, sheet, cur, buildCatRow(k), 2, styles)
	}

	// Compute monthly totals
	primaryByMonth := make(map[int]decimal.Decimal)
	secondaryByMonth := make(map[int]decimal.Decimal)
	for _, k := range inflows {
		for _, m := range months {
			primaryByMonth[m] = primaryByMonth[m].Add(catMonthly[k][m])
		}
	}
	for _, k := range outflows {
		for _, m := range months {
			secondaryByMonth[m] = secondaryByMonth[m].Add(catMonthly[k][m])
		}
	}
	effectiveByMonth := make(map[int]decimal.Decimal)
	for _, m := range months {
		effectiveByMonth[m] = primaryByMonth[m].Sub(secondaryByMonth[m])
	}

	buildSumRow := func(label string, byMonth map[int]decimal.Decimal) []string {
		vals := []string{label, ""}
		var total decimal.Decimal
		activeM := 0
		for _, m := range months {
			v := byMonth[m]
			vals = append(vals, v.StringFixed(2))
			total = total.Add(v)
			if !v.IsZero() {
				activeM++
			}
		}
		activeAvg := decimal.Zero
		if activeM > 0 {
			activeAvg = total.Div(decimal.NewFromInt(int64(activeM)))
		}
		calAvg := total.Div(decimal.NewFromInt(int64(calM)))
		return append(vals, total.StringFixed(2), activeAvg.StringFixed(2), calAvg.StringFixed(2))
	}

	sumHeaders := append(append([]string{"", ""}, monthCols...), "Total", "Avg/Active Month", "Avg/Calendar Month")

	// --- Monthly Summary ---
	cur++
	cur = xlsxTitle(f, sheet, cur, "Monthly Summary", styles.SectionTitle)
	cur++
	sumHeaderRow := cur
	cur = xlsxHeaderRow(f, sheet, cur, sumHeaders, styles)
	primarySumRow := cur
	cur = xlsxDataRow(f, sheet, cur, buildSumRow("Total Primary", primaryByMonth), 2, styles)
	secondarySumRow := cur
	cur = xlsxDataRow(f, sheet, cur, buildSumRow("Total Secondary", secondaryByMonth), 2, styles)
	effectiveSumRow := cur
	cur = xlsxDataRow(f, sheet, cur, buildSumRow("Effective (Primary - Secondary)", effectiveByMonth), 2, styles)

	momVals := []string{"MoM Change", ""}
	momPctVals := []string{"MoM % Change", ""}
	for i, m := range months {
		if i == 0 {
			momVals = append(momVals, "-")
			momPctVals = append(momPctVals, "-")
		} else {
			prev := effectiveByMonth[months[i-1]]
			curr := effectiveByMonth[m]
			diff := curr.Sub(prev)
			momVals = append(momVals, utils.SignedFixed(diff))
			if !prev.IsZero() {
				pct := diff.Div(prev.Abs()).Mul(decimal.NewFromInt(100))
				momPctVals = append(momPctVals, utils.SignedFixed(pct)+"%")
			} else {
				momPctVals = append(momPctVals, "-")
			}
		}
	}
	momVals = append(momVals, "", "", "")
	momPctVals = append(momPctVals, "", "", "")
	cur = xlsxDataRow(f, sheet, cur, momVals, 2, styles)
	cur = xlsxDataRow(f, sheet, cur, momPctVals, 2, styles)

	// --- Statistics ---
	effSlice := make([]decimal.Decimal, len(months))
	for i, m := range months {
		effSlice[i] = effectiveByMonth[m]
	}
	// Flip best/worst when only expense categories are selected in the primary slot
	// (no secondary): lower spending = better outcome.
	expenseOnly := len(outflows) == 0 && len(inflows) > 0
	for _, k := range inflows {
		if catDisplayClass[k] != "expense" {
			expenseOnly = false
			break
		}
	}
	bestVal, bestMonth := effSlice[0], months[0]
	worstVal, worstMonth := effSlice[0], months[0]
	var totalEff, totalPrimary, totalSecondary decimal.Decimal
	for i, v := range effSlice {
		totalEff = totalEff.Add(v)
		if expenseOnly {
			if v.LessThan(bestVal) {
				bestVal, bestMonth = v, months[i]
			}
			if v.GreaterThan(worstVal) {
				worstVal, worstMonth = v, months[i]
			}
		} else {
			if v.GreaterThan(bestVal) {
				bestVal, bestMonth = v, months[i]
			}
			if v.LessThan(worstVal) {
				worstVal, worstMonth = v, months[i]
			}
		}
	}
	for _, m := range months {
		totalPrimary = totalPrimary.Add(primaryByMonth[m])
		totalSecondary = totalSecondary.Add(secondaryByMonth[m])
	}
	avgEff := decimal.Zero
	if len(months) > 0 {
		avgEff = totalEff.Div(decimal.NewFromInt(int64(len(months))))
	}
	slope := utils.LinearTrend(effSlice)

	var totalTxnCount int
	for _, k := range inflows {
		for _, m := range months {
			totalTxnCount += catMonthlyCount[k][m]
		}
	}
	for _, k := range outflows {
		for _, m := range months {
			totalTxnCount += catMonthlyCount[k][m]
		}
	}
	avgTxnPerMonth := 0.0
	if len(months) > 0 {
		avgTxnPerMonth = float64(totalTxnCount) / float64(len(months))
	}

	cur++
	statsStartRow := cur
	cur = xlsxTitle(f, sheet, cur, "Statistics", styles.SectionTitle)
	cur++
	cur = xlsxHeaderRow(f, sheet, cur, []string{"Metric", "Value"}, styles)
	calAvgEff := totalEff.Div(decimal.NewFromInt(int64(calM)))

	for _, sr := range [][]string{
		{"Best Month", fmt.Sprintf("%s (%s)", monthAbbr[bestMonth-1], bestVal.StringFixed(2))},
		{"Worst Month", fmt.Sprintf("%s (%s)", monthAbbr[worstMonth-1], worstVal.StringFixed(2))},
		{"Median Month (Effective)", medianDecimal(effSlice).StringFixed(2)},
		{"Avg Effective / Active Month", avgEff.StringFixed(2)},
		{"Avg Effective / Calendar Month", calAvgEff.StringFixed(2)},
		{"Monthly Trend", fmt.Sprintf("%s per month (%s)", utils.SignedFixed(slope), utils.TrendDirection(slope))},
		{"Total Primary", totalPrimary.StringFixed(2)},
		{"Total Secondary", totalSecondary.StringFixed(2)},
		{"Total Effective", totalEff.StringFixed(2)},
		{"Total Transactions", fmt.Sprintf("%d", totalTxnCount)},
		{"Avg Transactions / Active Month", fmt.Sprintf("%.1f", avgTxnPerMonth)},
		{"Active Months", fmt.Sprintf("%d", len(months))},
		{"Calendar Months", fmt.Sprintf("%d", calM)},
	} {
		cur = xlsxDataRow(f, sheet, cur, sr, 1, styles)
	}

	primaryLabel := "Primary"
	if len(inflows) == 1 && len(outflows) == 0 {
		primaryLabel = inflows[0].name
	}
	j.addYearCharts(f, sheet, months, sumHeaderRow, primarySumRow, secondarySumRow, effectiveSumRow, statsStartRow, len(outflows) > 0, primaryLabel)

	if err := f.SetColWidth(sheet, "A", "A", 32); err != nil {
		return
	}
	if err := f.SetColWidth(sheet, "B", "B", 16); err != nil {
		return
	}
	for i := range months {
		col, _ := excelize.ColumnNumberToName(i + 3)
		if err := f.SetColWidth(sheet, col, col, 11); err != nil {
			return
		}
	}
	totalCol, _ := excelize.ColumnNumberToName(len(months) + 3)
	activeAvgCol, _ := excelize.ColumnNumberToName(len(months) + 4)
	calAvgCol, _ := excelize.ColumnNumberToName(len(months) + 5)
	if err := f.SetColWidth(sheet, totalCol, totalCol, 14); err != nil {
		return
	}
	if err := f.SetColWidth(sheet, activeAvgCol, activeAvgCol, 22); err != nil {
		return
	}
	if err := f.SetColWidth(sheet, calAvgCol, calAvgCol, 24); err != nil {
		return
	}
}

func (j *GenerateCategoryReportJob) writeAllTimeSheet(f *excelize.File, sheet string, styles xlsxStyles, rows []models.CategoryReportDataRow, years []int) {
	catYearly := make(map[catKey]map[int]decimal.Decimal)
	catYearlyCount := make(map[catKey]map[int]int)
	catDisplayClass := make(map[catKey]string)
	for _, r := range rows {
		k := catKey{r.CategoryName, r.Classification}
		if catYearly[k] == nil {
			catYearly[k] = make(map[int]decimal.Decimal)
			catYearlyCount[k] = make(map[int]int)
		}
		catYearly[k][r.Year] = catYearly[k][r.Year].Add(r.Total)
		catYearlyCount[k][r.Year] += r.TxnCount
		catDisplayClass[k] = r.CategoryClassification
	}

	var inflows, outflows []catKey
	for k := range catYearly {
		if k.classification == "inflow" {
			inflows = append(inflows, k)
		} else {
			outflows = append(outflows, k)
		}
	}
	sort.Slice(inflows, func(i, j int) bool { return inflows[i].name < inflows[j].name })
	sort.Slice(outflows, func(i, j int) bool { return outflows[i].name < outflows[j].name })

	yearStrs := utils.YearStrings(years)
	catHeaders := append(append([]string{"Category", "Classification"}, yearStrs...), "Total", "Avg/Year")

	buildCatRow := func(k catKey) []string {
		vals := []string{k.name, catDisplayClass[k]}
		var total decimal.Decimal
		for _, y := range years {
			v := catYearly[k][y]
			vals = append(vals, v.StringFixed(2))
			total = total.Add(v)
		}
		avg := total.Div(decimal.NewFromInt(int64(len(years))))
		return append(vals, total.StringFixed(2), avg.StringFixed(2))
	}

	cur := 1
	cur = xlsxTitle(f, sheet, cur, "All-Time Category Comparison", styles.SectionTitle)
	cur++
	cur = xlsxHeaderRow(f, sheet, cur, catHeaders, styles)
	for _, k := range inflows {
		cur = xlsxDataRow(f, sheet, cur, buildCatRow(k), 2, styles)
	}
	if len(inflows) > 0 && len(outflows) > 0 {
		cur++
	}
	for _, k := range outflows {
		cur = xlsxDataRow(f, sheet, cur, buildCatRow(k), 2, styles)
	}

	primaryByYear := make(map[int]decimal.Decimal)
	secondaryByYear := make(map[int]decimal.Decimal)
	for _, k := range inflows {
		for _, y := range years {
			primaryByYear[y] = primaryByYear[y].Add(catYearly[k][y])
		}
	}
	for _, k := range outflows {
		for _, y := range years {
			secondaryByYear[y] = secondaryByYear[y].Add(catYearly[k][y])
		}
	}
	effectiveByYear := make(map[int]decimal.Decimal)
	for _, y := range years {
		effectiveByYear[y] = primaryByYear[y].Sub(secondaryByYear[y])
	}

	buildYearRow := func(label string, byYear map[int]decimal.Decimal) []string {
		vals := []string{label, ""}
		var total decimal.Decimal
		for _, y := range years {
			v := byYear[y]
			vals = append(vals, v.StringFixed(2))
			total = total.Add(v)
		}
		avg := total.Div(decimal.NewFromInt(int64(len(years))))
		return append(vals, total.StringFixed(2), avg.StringFixed(2))
	}

	sumHeaders := append(append([]string{"", ""}, yearStrs...), "Total", "Avg/Year")

	cur++
	cur = xlsxTitle(f, sheet, cur, "Annual Summary", styles.SectionTitle)
	cur++
	annualHeaderRow := cur
	cur = xlsxHeaderRow(f, sheet, cur, sumHeaders, styles)
	cur = xlsxDataRow(f, sheet, cur, buildYearRow("Total Primary", primaryByYear), 2, styles)
	cur = xlsxDataRow(f, sheet, cur, buildYearRow("Total Secondary", secondaryByYear), 2, styles)
	effectiveYearRow := cur
	cur = xlsxDataRow(f, sheet, cur, buildYearRow("Effective", effectiveByYear), 2, styles)

	effSlice := make([]decimal.Decimal, len(years))
	for i, y := range years {
		effSlice[i] = effectiveByYear[y]
	}
	yoyRow := []string{"YoY Change (Effective)", "", "-"}
	yoyPctRow := []string{"YoY % Change", "", "-"}
	for i := 1; i < len(years); i++ {
		diff := effSlice[i].Sub(effSlice[i-1])
		yoyRow = append(yoyRow, utils.SignedFixed(diff))
		if !effSlice[i-1].IsZero() {
			pct := diff.Div(effSlice[i-1].Abs()).Mul(decimal.NewFromInt(100))
			yoyPctRow = append(yoyPctRow, utils.SignedFixed(pct)+"%")
		} else {
			yoyPctRow = append(yoyPctRow, "-")
		}
	}
	yoyRow = append(yoyRow, "", "")
	yoyPctRow = append(yoyPctRow, "", "")
	cur = xlsxDataRow(f, sheet, cur, yoyRow, 2, styles)
	cur = xlsxDataRow(f, sheet, cur, yoyPctRow, 2, styles)

	expenseOnly := len(outflows) == 0 && len(inflows) > 0
	for _, k := range inflows {
		if catDisplayClass[k] != "expense" {
			expenseOnly = false
			break
		}
	}
	bestVal, bestYear := effSlice[0], years[0]
	worstVal, worstYear := effSlice[0], years[0]
	var totalEff decimal.Decimal
	for i, v := range effSlice {
		totalEff = totalEff.Add(v)
		if expenseOnly {
			if v.LessThan(bestVal) {
				bestVal, bestYear = v, years[i]
			}
			if v.GreaterThan(worstVal) {
				worstVal, worstYear = v, years[i]
			}
		} else {
			if v.GreaterThan(bestVal) {
				bestVal, bestYear = v, years[i]
			}
			if v.LessThan(worstVal) {
				worstVal, worstYear = v, years[i]
			}
		}
	}
	avgEff := totalEff.Div(decimal.NewFromInt(int64(len(years))))
	slope := utils.LinearTrend(effSlice)

	var totalTxnCount int
	for _, k := range inflows {
		for _, y := range years {
			totalTxnCount += catYearlyCount[k][y]
		}
	}
	for _, k := range outflows {
		for _, y := range years {
			totalTxnCount += catYearlyCount[k][y]
		}
	}
	avgTxnPerYear := float64(totalTxnCount) / float64(len(years))

	cur++
	allTimeStatsRow := cur
	cur = xlsxTitle(f, sheet, cur, "Statistics", styles.SectionTitle)
	cur++
	cur = xlsxHeaderRow(f, sheet, cur, []string{"Metric", "Value"}, styles)
	for _, sr := range [][]string{
		{"Best Year", fmt.Sprintf("%d (%s)", bestYear, bestVal.StringFixed(2))},
		{"Worst Year", fmt.Sprintf("%d (%s)", worstYear, worstVal.StringFixed(2))},
		{"Median Year (Effective)", medianDecimal(effSlice).StringFixed(2)},
		{"Avg Annual Effective", avgEff.StringFixed(2)},
		{"Annual Trend", fmt.Sprintf("%s per year (%s)", utils.SignedFixed(slope), utils.TrendDirection(slope))},
		{"Total Transactions", fmt.Sprintf("%d", totalTxnCount)},
		{"Avg Transactions / Year", fmt.Sprintf("%.1f", avgTxnPerYear)},
		{"Years Covered", fmt.Sprintf("%d", len(years))},
	} {
		cur = xlsxDataRow(f, sheet, cur, sr, 1, styles)
	}

	primaryLabel := "Effective"
	if len(inflows) == 1 && len(outflows) == 0 {
		primaryLabel = inflows[0].name
	}
	j.addAllTimeChart(f, sheet, years, annualHeaderRow, effectiveYearRow, allTimeStatsRow, primaryLabel)

	if err := f.SetColWidth(sheet, "A", "A", 30); err != nil {
		return
	}
	if err := f.SetColWidth(sheet, "B", "B", 16); err != nil {
		return
	}
	for i := range years {
		col, _ := excelize.ColumnNumberToName(i + 3)
		if err := f.SetColWidth(sheet, col, col, 14); err != nil {
			return
		}
	}
	totalCol, _ := excelize.ColumnNumberToName(len(years) + 3)
	avgCol, _ := excelize.ColumnNumberToName(len(years) + 4)
	if err := f.SetColWidth(sheet, totalCol, totalCol, 14); err != nil {
		return
	}
	if err := f.SetColWidth(sheet, avgCol, avgCol, 14); err != nil {
		return
	}
}

func (j *GenerateCategoryReportJob) addYearCharts(
	f *excelize.File,
	sheet string,
	months []int,
	sumHeaderRow, primarySumRow, secondarySumRow, effectiveSumRow int,
	statsStartRow int,
	hasSecondary bool,
	primaryLabel string,
) {
	if len(months) == 0 {
		return
	}

	colStart, _ := excelize.ColumnNumberToName(3)
	colEnd, _ := excelize.ColumnNumberToName(2 + len(months))

	ref := func(row int) string {
		return fmt.Sprintf("'%s'!$%s$%d:$%s$%d", sheet, colStart, row, colEnd, row)
	}

	dim := excelize.ChartDimension{Width: 520, Height: 330}
	chartCol := len(months) + 7

	makeChart := func(title, name, valRef string) *excelize.Chart {
		return &excelize.Chart{
			Type:      excelize.Col,
			Dimension: dim,
			Legend:    excelize.ChartLegend{Position: "none"},
			PlotArea: excelize.ChartPlotArea{
				ShowVal: true,
			},
			Title: []excelize.RichTextRun{{Text: title}},
			Series: []excelize.ChartSeries{{
				Name:       name,
				Categories: ref(sumHeaderRow),
				Values:     valRef,
			}},
		}
	}

	if hasSecondary {
		a1, _ := excelize.CoordinatesToCellName(chartCol, statsStartRow)
		a2, _ := excelize.CoordinatesToCellName(chartCol, statsStartRow+18)
		a3, _ := excelize.CoordinatesToCellName(chartCol, statsStartRow+36)
		_ = f.AddChart(sheet, a1, makeChart("Effective per Month", "Effective", ref(effectiveSumRow)))
		_ = f.AddChart(sheet, a2, makeChart("Primary per Month", "Primary", ref(primarySumRow)))
		_ = f.AddChart(sheet, a3, makeChart("Secondary per Month", "Secondary", ref(secondarySumRow)))
	} else {
		a1, _ := excelize.CoordinatesToCellName(chartCol, statsStartRow)
		title := fmt.Sprintf("%s per Month", primaryLabel)
		_ = f.AddChart(sheet, a1, makeChart(title, primaryLabel, ref(primarySumRow)))
	}
}

func (j *GenerateCategoryReportJob) addAllTimeChart(
	f *excelize.File,
	sheet string,
	years []int,
	annualHeaderRow, effectiveYearRow int,
	statsStartRow int,
	primaryLabel string,
) {
	if len(years) == 0 {
		return
	}

	colStart, _ := excelize.ColumnNumberToName(3)
	colEnd, _ := excelize.ColumnNumberToName(2 + len(years))

	catRef := fmt.Sprintf("'%s'!$%s$%d:$%s$%d", sheet, colStart, annualHeaderRow, colEnd, annualHeaderRow)
	effRef := fmt.Sprintf("'%s'!$%s$%d:$%s$%d", sheet, colStart, effectiveYearRow, colEnd, effectiveYearRow)

	chartCol := len(years) + 7
	anchor, _ := excelize.CoordinatesToCellName(chartCol, statsStartRow)

	_ = f.AddChart(sheet, anchor, &excelize.Chart{
		Type:      excelize.Col,
		Dimension: excelize.ChartDimension{Width: 520, Height: 330},
		Legend:    excelize.ChartLegend{Position: "none"},
		PlotArea: excelize.ChartPlotArea{
			ShowVal: true,
		},

		Title: []excelize.RichTextRun{{Text: fmt.Sprintf("%s per Year", primaryLabel)}},
		Series: []excelize.ChartSeries{{
			Name:       primaryLabel,
			Categories: catRef,
			Values:     effRef,
		}},
	})
}

func (j *GenerateCategoryReportJob) reportName(categoryLabel string) string {
	var yearPart string
	if j.Params.AllTime {
		yearPart = "All Time"
	} else {
		parts := make([]string, len(j.Params.Years))
		for i, y := range j.Params.Years {
			parts[i] = fmt.Sprintf("%d", y)
		}
		yearPart = strings.Join(parts, ", ")
	}
	name := fmt.Sprintf("Category Report - %s - %s", categoryLabel, yearPart)
	if j.Params.Description != "" {
		name += fmt.Sprintf(" (%s)", j.Params.Description)
	}
	return name
}

func (j *GenerateCategoryReportJob) saveFile(data []byte) (string, error) {
	dir := filepath.Join("storage", "reports", fmt.Sprintf("%d", j.UserID))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	filePath := filepath.Join(dir, fmt.Sprintf("%d.xlsx", j.ReportID))
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", err
	}
	return filePath, nil
}

func sortedYears(m map[int][]models.CategoryReportDataRow) []int {
	years := make([]int, 0, len(m))
	for y := range m {
		years = append(years, y)
	}
	sort.Ints(years)
	return years
}

func deriveCategoryLabel(rows []models.CategoryReportDataRow) string {
	seen := make(map[string]struct{})
	for _, r := range rows {
		if r.Classification == "inflow" {
			seen[r.CategoryName] = struct{}{}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) > 0 {
		return names[0]
	}
	return "Unknown"
}

func xlsxTitle(f *excelize.File, sheet string, row int, title string, styleID int) int {
	cell, _ := excelize.CoordinatesToCellName(1, row)
	if err := f.SetCellValue(sheet, cell, title); err != nil {
		return row
	}
	if err := f.SetCellStyle(sheet, cell, cell, styleID); err != nil {
		return row
	}
	return row + 1
}

func xlsxHeaderRow(f *excelize.File, sheet string, row int, headers []string, styles xlsxStyles) int {
	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, row)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return row
		}
		if err := f.SetCellStyle(sheet, cell, cell, styles.ColHeader); err != nil {
			return row
		}
	}
	return row + 1
}

func medianDecimal(vals []decimal.Decimal) decimal.Decimal {
	if len(vals) == 0 {
		return decimal.Zero
	}
	sorted := make([]decimal.Decimal, len(vals))
	copy(sorted, vals)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].LessThan(sorted[j]) })
	n := len(sorted)
	if n%2 == 0 {
		return sorted[n/2-1].Add(sorted[n/2]).Div(decimal.NewFromInt(2))
	}
	return sorted[n/2]
}

// xlsxDataRow writes values; first labelCols use label style, rest use data style.
// Non-label columns that parse as float64 are written as numbers so Excel charts can plot them.
func xlsxDataRow(f *excelize.File, sheet string, row int, values []string, labelCols int, styles xlsxStyles) int {
	for col, v := range values {
		cell, _ := excelize.CoordinatesToCellName(col+1, row)
		if col >= labelCols {
			if fv, err := strconv.ParseFloat(v, 64); err == nil {
				if err := f.SetCellValue(sheet, cell, fv); err != nil {
					return row
				}
			} else {
				if err := f.SetCellValue(sheet, cell, v); err != nil {
					return row
				}
			}
			if err := f.SetCellStyle(sheet, cell, cell, styles.DataCell); err != nil {
				return row
			}
		} else {
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				return row
			}
			if err := f.SetCellStyle(sheet, cell, cell, styles.LabelCell); err != nil {
				return row
			}
		}
	}
	return row + 1
}
