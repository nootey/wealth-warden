import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import timezone from "dayjs/plugin/timezone";
import dateHelper from "./date_helper.ts";

dayjs.extend(utc);
dayjs.extend(timezone);

const withTZ = (tz: string, run: () => void): void => {
  const previous = process.env.TZ;
  process.env.TZ = tz;
  try {
    run();
  } finally {
    process.env.TZ = previous;
  }
};

describe("dateHelper", () => {
  describe("formatDate", () => {
    it("returns an empty string for a missing date", () => {
      expect(dateHelper.formatDate("")).toBe("");
    });

    it("never shifts a date-only string, whatever the host timezone", () => {
      for (const tz of ["America/New_York", "Asia/Tokyo", "UTC"]) {
        withTZ(tz, () => {
          expect(dateHelper.formatDate("2026-07-09")).toBe("2026-07-09");
        });
      }
    });

    it("renders a timestamp in the host timezone", () => {
      withTZ("America/New_York", () => {
        expect(dateHelper.formatDate("2026-07-09T02:00:00Z")).toBe(
          "2026-07-08",
        );
      });
      withTZ("Asia/Tokyo", () => {
        expect(dateHelper.formatDate("2026-07-09T02:00:00Z")).toBe(
          "2026-07-09",
        );
      });
    });

    it("renders the utc calendar date when asked", () => {
      withTZ("America/New_York", () => {
        expect(
          dateHelper.formatDate(
            "2026-07-09T02:00:00Z",
            false,
            "YYYY-MM-DD",
            true,
          ),
        ).toBe("2026-07-09");
      });
    });

    it("appends the time when asked", () => {
      withTZ("UTC", () => {
        expect(dateHelper.formatDate("2026-07-09T02:30:00Z", true)).toBe(
          "2026-07-09 02:30",
        );
      });
    });

    it("honours a custom format", () => {
      expect(dateHelper.formatDate("2026-07-09", false, "DD/MM/YYYY")).toBe(
        "09/07/2026",
      );
    });
  });

  describe("mergeDateWithCurrentTime", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it("preserves the picked calendar date in the target timezone", () => {
      vi.setSystemTime(new Date("2026-07-09T23:30:00Z"));

      withTZ("Asia/Tokyo", () => {
        for (const tz of [
          "UTC",
          "America/New_York",
          "Asia/Tokyo",
          "Pacific/Kiritimati",
          "Pacific/Midway",
        ]) {
          const iso = dateHelper.mergeDateWithCurrentTime("2026-07-09", tz);
          expect(dayjs(iso).tz(tz).format("YYYY-MM-DD")).toBe("2026-07-09");
        }
      });
    });

    it("stamps the current wall-clock time of the target timezone", () => {
      vi.setSystemTime(new Date("2026-07-09T23:30:15Z"));

      const iso = dateHelper.mergeDateWithCurrentTime(
        "2026-07-09",
        "Asia/Tokyo",
      );

      expect(dayjs(iso).tz("Asia/Tokyo").format("HH:mm:ss")).toBe("08:30:15");
    });

    it("defaults to utc", () => {
      vi.setSystemTime(new Date("2026-07-09T23:30:00Z"));

      expect(dateHelper.mergeDateWithCurrentTime("2026-07-09")).toBe(
        "2026-07-09T23:30:00.000Z",
      );
    });

    it("preserves the picked date across a dst spring-forward", () => {
      vi.setSystemTime(new Date("2026-03-08T07:30:00Z"));

      const iso = dateHelper.mergeDateWithCurrentTime(
        "2026-03-08",
        "America/New_York",
      );

      expect(dayjs(iso).tz("America/New_York").format("YYYY-MM-DD")).toBe(
        "2026-03-08",
      );
    });

    it("rejects a date it cannot parse at all", () => {
      expect(() => dateHelper.mergeDateWithCurrentTime("garbage")).toThrow(
        /Invalid date format/,
      );
    });

    it("rejects a parseable date that is not YYYY-MM-DD", () => {
      for (const bad of ["07/09/2026", "2026-7-9", "July 9, 2026"]) {
        expect(() => dateHelper.mergeDateWithCurrentTime(bad)).toThrow(
          /Invalid date format/,
        );
      }
    });
  });

  describe("convertTimeZone", () => {
    it("formats in the given timezone", () => {
      expect(
        dateHelper.convertTimeZone("2026-07-09T02:00:00Z", "America/New_York"),
      ).toBe("08-07-2026 22:00");
    });

    it("includes seconds when asked", () => {
      expect(
        dateHelper.convertTimeZone("2026-07-09T02:00:00Z", "UTC", true),
      ).toBe("09-07-2026 02:00:00");
    });

    it("omits the time when asked", () => {
      expect(
        dateHelper.convertTimeZone("2026-07-09T02:00:00Z", "UTC", false, false),
      ).toBe("09-07-2026");
    });

    it("falls back to the host timezone when none is given", () => {
      withTZ("Asia/Tokyo", () => {
        expect(dateHelper.convertTimeZone("2026-07-09T02:00:00Z")).toBe(
          "09-07-2026 11:00",
        );
      });
    });
  });

  describe("combineDateAndTime", () => {
    it("takes the date from the first source and the time from the second", () => {
      expect(
        dateHelper.combineDateAndTime(
          "2026-07-09T00:00:00Z",
          "2020-01-01T13:45:30Z",
          "UTC",
        ),
      ).toBe("2026-07-09 13:45");
    });

    it("resolves both sources in the given timezone", () => {
      expect(
        dateHelper.combineDateAndTime(
          "2026-07-09T00:00:00Z",
          "2020-01-01T13:45:30Z",
          "Asia/Tokyo",
        ),
      ).toBe("2026-07-09 22:45");
    });

    it("returns an empty string when either source is missing", () => {
      expect(dateHelper.combineDateAndTime("", "2020-01-01T13:45:30Z")).toBe(
        "",
      );
      expect(dateHelper.combineDateAndTime("2026-07-09T00:00:00Z", "")).toBe(
        "",
      );
    });
  });
});
