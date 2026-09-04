import { PageHeader } from "@/components/prova/page-header"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { auditEntries } from "./mock"

export default function AuditLogPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="Denetim kaydı"
        description="Değiştirilemez kayıt. Puan ezme, sürüm yayınlama ve veri silme işlemleri buraya düşer."
      />

      <div className="min-w-0 rounded-lg border border-border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[180px]">Zaman</TableHead>
              <TableHead className="w-[280px]">Aktör</TableHead>
              <TableHead>Eylem</TableHead>
              <TableHead className="w-[200px]">Hedef</TableHead>
              <TableHead className="w-[140px]">Kaynak IP</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {auditEntries.map((row) => (
              <TableRow key={row.id}>
                <TableCell className="prova-meta">{row.time}</TableCell>
                <TableCell className="text-muted-foreground">
                  {row.actor}
                </TableCell>
                <TableCell>{row.action}</TableCell>
                <TableCell className="font-mono text-xs">
                  {row.target}
                </TableCell>
                <TableCell className="font-mono text-xs text-muted-foreground">
                  {row.sourceIp}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
