"use client"

import * as React from "react"
import { Plus } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { assignableEmployees, assignableScenarios } from "./mock"

const employeeOptions = Object.fromEntries(
  assignableEmployees.map((name) => [name, name])
)

const scenarioOptions = Object.fromEntries(
  assignableScenarios.map((scenario) => [
    scenario.name,
    `${scenario.name} · ${scenario.version}`,
  ])
)

export function AssignmentDialog({
  onAssign,
}: {
  onAssign: (input: { employee: string; scenario: string; dueDate: string }) => void
}) {
  const [open, setOpen] = React.useState(false)
  const [employee, setEmployee] = React.useState(assignableEmployees[0])
  const [scenario, setScenario] = React.useState(assignableScenarios[0].name)
  const [dueDate, setDueDate] = React.useState("")

  function submit(event: React.FormEvent) {
    event.preventDefault()
    onAssign({ employee, scenario, dueDate })
    toast.success("Senaryo atandı", {
      description: `${employee} · ${scenario}`,
    })
    setDueDate("")
    setOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button size="lg" />}>
        <Plus aria-hidden />
        Senaryo ata
      </DialogTrigger>
      <DialogContent className="sm:max-w-[440px]">
        <form onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>Senaryo ata</DialogTitle>
            <DialogDescription>
              Atanan senaryo, çalışanın masaüstü uygulamasında görünür.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-5">
            <div className="space-y-2">
              <Label>Çalışan</Label>
              <Select
                items={employeeOptions}
                value={employee}
                onValueChange={(value) => setEmployee(value as string)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {assignableEmployees.map((name) => (
                    <SelectItem key={name} value={name}>
                      {name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Senaryo</Label>
              <Select
                items={scenarioOptions}
                value={scenario}
                onValueChange={(value) => setScenario(value as string)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {assignableScenarios.map((item) => (
                    <SelectItem key={item.name} value={item.name}>
                      {item.name} · {item.version}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="prova-meta normal-case">
                Oturum, atama anındaki yayında olan sürümle oynanır.
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="assignment-due-date">Son tarih</Label>
              <Input
                id="assignment-due-date"
                value={dueDate}
                onChange={(event) => setDueDate(event.target.value)}
                placeholder="GG.AA.YYYY"
                required
              />
            </div>
          </div>

          <DialogFooter>
            <DialogClose render={<Button variant="outline" type="button" />}>
              Vazgeç
            </DialogClose>
            <Button type="submit">Ata</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
