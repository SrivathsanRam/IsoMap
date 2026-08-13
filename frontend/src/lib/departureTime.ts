export const DEFAULT_ISOCHRONE_DEPARTURE_TIME = "08:00";
export const TIME_OF_DAY_STEP_MINUTES = 5;
export const MINUTES_PER_DAY = 24 * 60;
export const MAX_TIME_OF_DAY_MINUTES = MINUTES_PER_DAY - TIME_OF_DAY_STEP_MINUTES;

const singaporeTimeZone = "Asia/Singapore";
const validDepartureTime = /^(?:[01]\d|2[0-3]):[0-5]\d$/;

export function departureTimeToMinutes(time: string) {
  if (!validDepartureTime.test(time)) {
    throw new Error("Departure time must be a valid 24-hour time");
  }

  const [hours, minutes] = time.split(":").map(Number);
  return hours * 60 + minutes;
}

export function minutesToDepartureTime(minutes: number) {
  if (!Number.isInteger(minutes) || minutes < 0 || minutes >= MINUTES_PER_DAY) {
    throw new Error("Departure minutes must be an integer between 0 and 1439");
  }

  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return `${String(hours).padStart(2, "0")}:${String(remainingMinutes).padStart(2, "0")}`;
}

export function normalizeTimeOfDayMinutes(
  minutes: number,
  stepMinutes = TIME_OF_DAY_STEP_MINUTES,
) {
  if (!Number.isFinite(minutes)) {
    throw new Error("Departure minutes must be a finite number");
  }
  if (!Number.isInteger(stepMinutes) || stepMinutes <= 0 || stepMinutes > MINUTES_PER_DAY) {
    throw new Error("Time step must be a positive integer no greater than one day");
  }

  const lastStep = Math.floor((MINUTES_PER_DAY - 1) / stepMinutes) * stepMinutes;
  return Math.min(lastStep, Math.max(0, Math.round(minutes / stepMinutes) * stepMinutes));
}

export function formatDepartureTime(time: string) {
  const totalMinutes = departureTimeToMinutes(time);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  const period = hours < 12 ? "AM" : "PM";
  const displayHours = hours % 12 || 12;
  return `${displayHours}:${String(minutes).padStart(2, "0")} ${period}`;
}

export function toSingaporeDepartureRFC3339(time: string, now = new Date()) {
  departureTimeToMinutes(time);

  const dateParts = new Intl.DateTimeFormat("en-CA", {
    timeZone: singaporeTimeZone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(now);
  const part = (type: "year" | "month" | "day") =>
    dateParts.find((candidate) => candidate.type === type)?.value;
  const year = part("year");
  const month = part("month");
  const day = part("day");
  if (!year || !month || !day) {
    throw new Error("Could not determine the current Singapore service date");
  }

  return `${year}-${month}-${day}T${time}:00+08:00`;
}
