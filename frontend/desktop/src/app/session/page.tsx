"use client"

import * as React from "react"
import Link from "next/link"

import { Button } from "@/components/ui/button"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { TopBar } from "@/components/top-bar"
import { ConnectionStatus } from "@/components/session/connection-status"
import { ConnectionOverlay } from "@/components/session/connection-overlay"
import { PushToTalk } from "@/components/session/push-to-talk"
import { RubricPanel } from "@/components/session/rubric-panel"
import { Transcript } from "@/components/session/transcript"
import { useSession } from "@/hooks/use-session"
import { sessionCriteria } from "./mock"

export default function SessionPage() {
  const session = useSession()
  const { startSpeaking, stopSpeaking } = session

  // Space is captured globally for push-to-talk; no button is ever triggered by it.
  React.useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (event.code !== "Space") return
      event.preventDefault()
      if (event.repeat) return
      startSpeaking()
    }

    function handleKeyUp(event: KeyboardEvent) {
      if (event.code !== "Space") return
      event.preventDefault()
      stopSpeaking()
    }

    function handleBlur() {
      stopSpeaking()
    }

    window.addEventListener("keydown", handleKeyDown, { capture: true })
    window.addEventListener("keyup", handleKeyUp, { capture: true })
    window.addEventListener("blur", handleBlur)
    return () => {
      window.removeEventListener("keydown", handleKeyDown, { capture: true })
      window.removeEventListener("keyup", handleKeyUp, { capture: true })
      window.removeEventListener("blur", handleBlur)
    }
  }, [startSpeaking, stopSpeaking])

  const minutes = String(Math.floor(session.elapsedSeconds / 60)).padStart(2, "0")
  const seconds = String(session.elapsedSeconds % 60).padStart(2, "0")

  return (
    <div
      className="prova-sinav flex h-full flex-col overflow-hidden"
      onContextMenu={(event) => event.preventDefault()}
      onCopy={(event) => event.preventDefault()}
    >
      <TopBar sade />

      <div className="flex min-h-0 flex-1 flex-col gap-4 p-5">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="font-heading text-xl tracking-tight">
              {session.scenario.name}
            </h1>
            <p className="prova-meta normal-case">
              {session.scenario.personaName} · sürüm {session.scenario.version} ·
              oturum {session.scenario.sessionNo}
            </p>
          </div>
          <div className="flex items-center gap-3">
            <span className="prova-meta font-mono">
              {minutes}:{seconds}
            </span>
            <ConnectionStatus status={session.connection} />
            {!session.finished && (
              <AlertDialog>
                <AlertDialogTrigger render={<Button variant="outline" size="sm" />}>
                  Oturumu bitir
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>Oturum bitirilsin mi?</AlertDialogTitle>
                    <AlertDialogDescription>
                      Görüşme burada kapanır ve o ana kadarki transkript
                      değerlendirmeye gönderilir. Erken bitirilen oturum da
                      puanlanır.
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>Devam et</AlertDialogCancel>
                    <AlertDialogAction onClick={session.endSession}>
                      Oturumu bitir
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            )}
          </div>
        </div>

        <div className="flex min-h-0 flex-1 gap-4">
          <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-4">
            <div className="relative flex min-h-0 flex-1 flex-col">
              <Transcript
                lines={session.lines}
                employeeName={session.scenario.employee}
                characterName={session.scenario.personaName}
              />
              {session.connection === "disconnected" ? <ConnectionOverlay /> : null}
            </div>

            {session.finished ? (
              <div className="flex items-center justify-between rounded-lg border border-line-strong bg-card px-4 py-3">
                <p className="text-sm text-foreground">
                  Görüşme tamamlandı. Değerlendirme güçlü modele iletildi.
                </p>
                <Button nativeButton={false} render={<Link href="/result" />}>Sonucu gör</Button>
              </div>
            ) : (
              <PushToTalk
                speaking={session.speaking}
                characterReplying={session.characterReplying}
                finished={session.finished}
              />
            )}
          </div>

          <RubricPanel criteria={sessionCriteria} rubric={session.rubric} />
        </div>
      </div>
    </div>
  )
}
