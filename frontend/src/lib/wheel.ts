export interface WeightedWheelSlice {
  probability: number;
  color: string;
}

export interface ClientWheelSegment extends WeightedWheelSlice {
  prize_id: string;
  label: string;
  prize_type: string;
  base_value: number;
  is_active: boolean;
}

export interface WheelSliceRange {
  start: number;
  end: number;
  center: number;
  sweep: number;
}

export function mapWheelSlots(raw: Record<string, unknown>[]): ClientWheelSegment[] {
  return raw
    .filter((p) => p.is_active !== false)
    .map((p) => ({
      prize_id: String(p.prize_id ?? p.id ?? ""),
      label: String(p.label ?? p.name ?? p.prize_name ?? "Prize"),
      prize_type: String(p.prize_type ?? p.type ?? "try_again").toLowerCase(),
      base_value: Number(p.base_value ?? p.prize_value ?? p.value ?? 0),
      probability: Number(p.probability ?? p.win_probability_weight ?? 0),
      color: String(
        p.prize_type === "try_again" || p.is_no_win
          ? (p.color ?? "#374151")
          : (p.color ?? p.color_hex ?? "#5f72f9"),
      ),
      is_active: true,
    }));
}

export function wheelTotalCents<T extends WeightedWheelSlice>(segments: T[]): number {
  return segments.reduce(
    (sum, segment) => sum + Math.round(segment.probability * 100),
    0,
  );
}

export function isValidWheelPercentages<T extends WeightedWheelSlice>(segments: T[]): boolean {
  return (
    segments.length > 0 &&
    segments.every((segment) => segment.probability > 0) &&
    wheelTotalCents(segments) === 10000
  );
}

export const isValidPublishedWheel = isValidWheelPercentages;

export function buildWheelSlices<T extends WeightedWheelSlice>(segments: T[]): WheelSliceRange[] {
  let cursor = 0;
  return segments.map((segment) => {
    const start = cursor;
    const sweep = segment.probability * 3.6;
    const end = start + sweep;
    const center = start + sweep / 2;
    cursor = end;
    return { start, end, center, sweep };
  });
}

export function buildWheelGradient<T extends WeightedWheelSlice>(segments: T[]): string {
  const slices = buildWheelSlices(segments);
  return `conic-gradient(${segments
    .map((segment, index) => {
      const slice = slices[index];
      return `${segment.color} ${slice.start}deg ${slice.end}deg`;
    })
    .join(", ")})`;
}

export const wheelConicGradient = buildWheelGradient;

export function segmentStartDeg<T extends WeightedWheelSlice>(segments: T[], index: number): number {
  return buildWheelSlices(segments)[index]?.start ?? 0;
}

export function segmentSweepDeg<T extends WeightedWheelSlice>(segments: T[], index: number): number {
  return buildWheelSlices(segments)[index]?.sweep ?? 0;
}

export function segmentCenterDeg<T extends WeightedWheelSlice>(segments: T[], index: number): number {
  return buildWheelSlices(segments)[index]?.center ?? 0;
}

export function wheelLandingDegrees<T extends WeightedWheelSlice>(
  segments: T[],
  index: number,
  pointerDegrees = 0,
): { landing: number; sweep: number } {
  const slice = buildWheelSlices(segments)[index];
  if (!slice) return { landing: 0, sweep: 0 };
  const landing = ((pointerDegrees - slice.center) % 360 + 360) % 360;
  return { landing, sweep: slice.sweep };
}
