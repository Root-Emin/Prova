import { PageHeader } from "@/components/prova/page-header"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { RoutingChart } from "./routing-chart"
import { modelUsages, routings } from "./mock"

export default function LlmRoutingPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="LLM dağılımı"
        description="Konuşmayı hızlı model yürütür, değerlendirmeyi güçlü model yapar. Yönlendirici gerektiğinde güçlü modele geçer."
      />

      <RoutingChart data={modelUsages} />

      <div className="min-w-0 rounded-lg border border-border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[220px]">Model</TableHead>
              <TableHead className="w-[140px]">Görev</TableHead>
              <TableHead className="w-[120px]">Oturum</TableHead>
              <TableHead>Ortalama gecikme</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {modelUsages.map((line) => (
              <TableRow key={`${line.model}-${line.task}`}>
                <TableCell className="font-mono text-xs">
                  {line.model}
                </TableCell>
                <TableCell>{line.task}</TableCell>
                <TableCell>{line.session}</TableCell>
                <TableCell className="prova-meta">
                  {line.avgLatencyMs} ms
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <section className="space-y-3">
        <h2 className="font-heading text-lg">Güçlü modele geçişler</h2>
        <div className="min-w-0 rounded-lg border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[180px]">Zaman</TableHead>
                <TableHead className="w-[120px]">Oturum</TableHead>
                <TableHead>Tetikleyen kural</TableHead>
                <TableHead className="w-[240px]">Hedef model</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {routings.map((row) => (
                <TableRow key={row.id}>
                  <TableCell className="prova-meta">{row.time}</TableCell>
                  <TableCell className="font-mono text-xs">
                    {row.session}
                  </TableCell>
                  <TableCell>{row.rule}</TableCell>
                  <TableCell className="font-mono text-xs">
                    {row.targetModel}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </section>
    </div>
  )
}
