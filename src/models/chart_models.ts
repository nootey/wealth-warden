export type ChartPoint = {
    date: string;
    value: number | string
}

export type NetworthResponse = {
    currency: string
    points: ChartPoint[]
    current: ChartPoint
}