import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import timezone from "dayjs/plugin/timezone";
import { computed } from "vue";

dayjs.extend(utc);
dayjs.extend(timezone);

const dateHelper = {
  monthColumns: computed(() => {
    return [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12];
  }),
  convertTimeZone(
    timestamp: string | number | Date,
    tz: string | null = null,
    showSeconds: boolean = false,
    showTime: boolean = true,
  ): string {
    let format = "DD-MM-YYYY";
    if (showTime) format += " HH:mm";
    if (showSeconds) format += ":ss";

    return tz
      ? dayjs.utc(timestamp).tz(tz).format(format)
      : dayjs.utc(timestamp).local().format(format);
  },

  formatDate(
    date_string: string | number | Date,
    time: boolean = false,
    date_format: string = "YYYY-MM-DD",
    utc: boolean = false,
  ): string {
    if (!date_string) return "";

    const isDateOnly =
      typeof date_string === "string" &&
      /^\d{4}-\d{2}-\d{2}$/.test(date_string);

    let date;

    if (isDateOnly) {
      date = dayjs(date_string);
    } else {
      date = utc ? dayjs.utc(date_string) : dayjs.utc(date_string).local();
    }

    const formatted_date = date.format(date_format);
    return time ? `${formatted_date} ${date.format("HH:mm")}` : formatted_date;
  },

  mergeDateWithCurrentTime(dateString: string, tz: string = "UTC"): string {
    const datePart = dayjs(dateString, "YYYY-MM-DD", true);
    if (!datePart.isValid()) {
      throw new Error(`Invalid date format: ${dateString}`);
    }

    const currentTime = dayjs.tz(new Date(), tz);

    const mergedDateTime = dayjs
      .tz(datePart.format("YYYY-MM-DD"), tz)
      .hour(currentTime.hour())
      .minute(currentTime.minute())
      .second(currentTime.second());

    return mergedDateTime.toISOString();
  },

  formatMonth(monthNumber: number): string {
    const currentYear = new Date().getFullYear();
    const date = new Date(currentYear, monthNumber - 1);
    return new Intl.DateTimeFormat(navigator.language, {
      month: "long",
    }).format(date);
  },

  mightBeDate(field: string | null): boolean {
    if (!field) return false;
    const dateFields: string[] = [];

    return (
      dateFields.includes(field.toLowerCase()) ||
      field.toLowerCase().includes("date")
    );
  },

  combineDateAndTime(
    dateSource: string | number | Date,
    timeSource: string | number | Date,
    timezone?: string,
    format: string = "YYYY-MM-DD HH:mm",
  ): string {
    if (!dateSource || !timeSource) return "";

    const tz = timezone || dayjs.tz.guess();

    const dateObj = dayjs.utc(dateSource).tz(tz);
    const timeObj = dayjs.utc(timeSource).tz(tz);

    const combined = dateObj
      .hour(timeObj.hour())
      .minute(timeObj.minute())
      .second(timeObj.second());

    return combined.format(format);
  },
};

export default dateHelper;
