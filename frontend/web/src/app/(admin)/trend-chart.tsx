"use client"

import * as React from "react"

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { PanelCard } from "./panel-card"
import { trendRanges, weeklyTrend } from "./mock"

/* Çizim alanı. Grafik kütüphanesi kurulmaz; iki seri de düz SVG'dir.
   Ölçüler viewBox birimidir, kart genişliğine göre ölçeklenir. */
const GENISLIK = 520
const YUKSEKLIK = 200
const SOL = 32
const SAG = 480
const UST = 12
const TABAN = 164

/** Sol eksen 0'dan başlar ve dört eşit adıma bölünür. */
function eksenTavani(enBuyuk: number) {
  const adim = Math.max(5, Math.ceil(enBuyuk / 4 / 5) * 5)
  return { adim, tavan: adim * 4 }
}

export function TrendChart() {
  const [aralik, setAralik] = React.useState<string>("8")

  const noktalar = weeklyTrend.slice(-Number(aralik))
  const { adim, tavan } = eksenTavani(
    Math.max(...noktalar.map((nokta) => nokta.sessions))
  )

  const yukseklik = TABAN - UST
  const dilim = (SAG - SOL) / noktalar.length
  const cubukGenisligi = Math.min(24, dilim * 0.46)
  const merkez = (index: number) => SOL + dilim * index + dilim / 2
  const oturumY = (deger: number) => TABAN - (deger / tavan) * yukseklik
  const oranY = (yuzde: number) => TABAN - (yuzde / 100) * yukseklik

  const cizgi = noktalar
    .map((nokta, index) => `${merkez(index)},${oranY(nokta.passRate)}`)
    .join(" ")

  const ilk = noktalar[0]
  const son = noktalar[noktalar.length - 1]

  return (
    <PanelCard
      title="Kurumsal performans özeti"
      description={`${noktalar.length} haftada oturum hacmi ve geçme oranı eğilimi`}
      className="lg:col-span-3"
      bodyClassName="flex flex-col justify-between gap-4"
      action={
        <Select
          items={Object.fromEntries(
            trendRanges.map((secenek) => [secenek.value, secenek.label])
          )}
          value={aralik}
          onValueChange={(value) => setAralik(value as string)}
        >
          <SelectTrigger size="sm" className="w-[132px] shrink-0 bg-card">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {trendRanges.map((secenek) => (
              <SelectItem key={secenek.value} value={secenek.value}>
                {secenek.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      }
    >
      <svg
        viewBox={`0 0 ${GENISLIK} ${YUKSEKLIK}`}
        className="h-auto w-full"
        role="img"
        aria-label={`Haftalık oturum sayısı ${ilk.sessions} ile ${son.sessions} arasında değişti, geçme oranı ${ilk.week} haftasında %${ilk.passRate} iken ${son.week} haftasında %${son.passRate} oldu.`}
      >
        {[0, 1, 2, 3, 4].map((sira) => {
          const y = TABAN - (sira / 4) * yukseklik
          return (
            <g key={sira}>
              <line
                x1={SOL}
                x2={SAG}
                y1={y}
                y2={y}
                className="stroke-border"
                strokeWidth={1}
              />
              <text
                x={SOL - 6}
                y={y + 3}
                textAnchor="end"
                fontSize={10}
                className="fill-muted-foreground"
              >
                {adim * sira}
              </text>
              <text
                x={SAG + 6}
                y={y + 3}
                textAnchor="start"
                fontSize={10}
                className="fill-muted-foreground"
              >
                %{sira * 25}
              </text>
            </g>
          )
        })}

        {noktalar.map((nokta, index) => (
          <rect
            key={`${nokta.week}-cubuk`}
            x={merkez(index) - cubukGenisligi / 2}
            y={oturumY(nokta.sessions)}
            width={cubukGenisligi}
            height={TABAN - oturumY(nokta.sessions)}
            rx={2}
            className="fill-tint"
          />
        ))}

        <polyline
          points={cizgi}
          fill="none"
          strokeWidth={1.5}
          strokeLinejoin="round"
          strokeLinecap="round"
          className="stroke-primary"
        />
        {noktalar.map((nokta, index) => (
          <circle
            key={`${nokta.week}-nokta`}
            cx={merkez(index)}
            cy={oranY(nokta.passRate)}
            r={2.5}
            className="fill-primary"
          />
        ))}

        {noktalar.map((nokta, index) => (
          <text
            key={`${nokta.week}-etiket`}
            x={merkez(index)}
            y={TABAN + 18}
            textAnchor="middle"
            fontSize={10}
            className="fill-muted-foreground"
          >
            {nokta.week}
          </text>
        ))}
      </svg>

      <p className="prova-meta flex flex-wrap items-center gap-4 normal-case">
        <span className="flex items-center gap-2">
          <span className="h-2 w-4 rounded-sm bg-tint" aria-hidden />
          Oturum sayısı
        </span>
        <span className="flex items-center gap-2">
          <span className="h-0.5 w-4 rounded-full bg-primary" aria-hidden />
          Geçme oranı
        </span>
      </p>
    </PanelCard>
  )
}
