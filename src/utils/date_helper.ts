import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import timezone from "dayjs/plugin/timezone";
import {computed} from "vue";

dayjs.extend(utc);
dayjs.extend(timezone);

const dateHelper = {
    monthColumns: computed(() => {
        return [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
    }),
    convertTimeZone(
        timestamp: string | number | Date,
        tz: string | null = null,
        showSeconds: boolean = false,
        showTime: boolean = true
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
        utc: boolean = false
    ): string {
        if (!date_string) return "";

        const date = utc ? dayjs.utc(date_string) : dayjs(date_string);
        const formatted_date = date.format(date_format);

        return time ? `${formatted_date} ${date.format("HH:mm")}` : formatted_date;
    },

    mergeDateWithCurrentTime(dateString: string, tz: string = "UTC"): string {

        const datePart = dayjs(dateString, 'YYYY-MM-DD', true);
        if (!datePart.isValid()) {
            throw new Error(`Invalid date format: ${dateString}`);
        }

        // Convert to timezone properly
        const dateWithTimezone = dayjs.tz(datePart.format('YYYY-MM-DD'), tz);
        if (!dateWithTimezone.isValid()) {
            throw new Error(`Invalid timezone parsing: ${dateWithTimezone}`);
        }

        // Get current time in specified timezone
        const currentTime = dayjs.tz(new Date(), tz);

        // Merge date with current time
        const mergedDateTime = dateWithTimezone
            .hour(currentTime.hour())
            .minute(currentTime.minute())
            .second(0);

        // Convert to UTC before returning
        return mergedDateTime.utc().toISOString();
    },

    formatMonth(monthNumber: number): string {
        const currentYear = new Date().getFullYear();
        const date = new Date(currentYear, monthNumber - 1);
        return new Intl.DateTimeFormat(navigator.language, { month: 'long' }).format(date)
    },

    mightBeDate(field: string | null): boolean {
        if (!field) return false;
        const dateFields = [];

        return (
            dateFields.includes(field.toLowerCase()) ||
            field.toLowerCase().includes("date")
        );
    }
};

export default dateHelper;