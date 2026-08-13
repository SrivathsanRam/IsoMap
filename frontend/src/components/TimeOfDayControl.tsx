import { useId, useState } from "react";
import {
  MAX_TIME_OF_DAY_MINUTES,
  TIME_OF_DAY_STEP_MINUTES,
  departureTimeToMinutes,
  formatDepartureTime,
  minutesToDepartureTime,
  normalizeTimeOfDayMinutes,
} from "../lib/departureTime";

type TimeOfDayControlProps = {
  value: string;
  onChange: (departureTime: string) => void;
  id?: string;
  label?: string;
  timeZoneLabel?: string;
  disabled?: boolean;
};

export function TimeOfDayControl({
  value,
  onChange,
  id,
  label = "Departure time",
  timeZoneLabel = "Singapore",
  disabled = false,
}: TimeOfDayControlProps) {
  const generatedID = useId();
  const controlID = id ?? `time-of-day-${generatedID}`;
  const sliderID = `${controlID}-slider`;
  const directInputID = `${controlID}-input`;
  const outputID = `${controlID}-output`;
  const minutes = normalizeTimeOfDayMinutes(departureTimeToMinutes(value));
  const normalizedValue = minutesToDepartureTime(minutes);
  const [directValue, setDirectValue] = useState(normalizedValue);
  const formattedTime = formatDepartureTime(normalizedValue);
  const accessibleTime = `${formattedTime} ${timeZoneLabel} time`;

  function changeMinutes(nextMinutes: number) {
    const nextValue = minutesToDepartureTime(normalizeTimeOfDayMinutes(nextMinutes));
    setDirectValue(nextValue);
    onChange(nextValue);
  }

  function changeDirectInput(nextValue: string) {
    setDirectValue(nextValue);
    if (!/^(?:[01]\d|2[0-3]):[0-5]\d$/.test(nextValue)) {
      return;
    }

    changeMinutes(departureTimeToMinutes(nextValue));
  }

  function commitDirectInput() {
    setDirectValue(normalizedValue);
  }

  return (
    <div className="time-of-day-control">
      <div className="time-of-day-header">
        <label htmlFor={sliderID}>{label}</label>
        <output id={outputID} htmlFor={sliderID} aria-live="polite">
          <strong>{formattedTime}</strong>
          <small>{timeZoneLabel}</small>
        </output>
      </div>

      <input
        id={sliderID}
        className="time-of-day-range"
        type="range"
        min="0"
        max={MAX_TIME_OF_DAY_MINUTES}
        step={TIME_OF_DAY_STEP_MINUTES}
        value={minutes}
        disabled={disabled}
        aria-valuetext={accessibleTime}
        aria-describedby={outputID}
        onChange={(event) => changeMinutes(Number(event.target.value))}
      />

      <div className="time-of-day-scale" aria-hidden="true">
        <span>12 AM</span>
        <span>12 PM</span>
        <span>11:55 PM</span>
      </div>

      <label className="time-of-day-direct" htmlFor={directInputID}>
        <span>Enter time</span>
        <input
          id={directInputID}
          type="time"
          step={TIME_OF_DAY_STEP_MINUTES * 60}
          value={directValue}
          disabled={disabled}
          onInput={(event) => changeDirectInput(event.currentTarget.value)}
          onBlur={commitDirectInput}
        />
      </label>
    </div>
  );
}
