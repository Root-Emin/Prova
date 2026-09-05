"use client"

import * as React from "react"

import type { RubricStatus } from "@/lib/rubric"
import {
  sessionCriteria,
  scenarioInfo,
  scenarioScript,
  type TranscriptLine,
} from "@/app/session/mock"

export type ConnectionState = "connected" | "reconnecting" | "disconnected"

export type SessionState = {
  scenario: typeof scenarioInfo
  lines: TranscriptLine[]
  rubric: Record<string, RubricStatus>
  connection: ConnectionState
  /** Space held down: the microphone is open. */
  speaking: boolean
  /** Waiting on the character's reply. */
  characterReplying: boolean
  elapsedSeconds: number
  finished: boolean
  startSpeaking: () => void
  stopSpeaking: () => void
  endSession: () => void
}

function formatTime(seconds: number) {
  const minutes = Math.floor(seconds / 60)
  const remaining = seconds % 60
  return `${String(minutes).padStart(2, "0")}:${String(remaining).padStart(2, "0")}`
}

/**
 * Live session state. The production path will feed local
 * Whisper Large V3 Turbo/whisper.cpp transcripts into the GraphQL session;
 * this renderer slice currently uses scripted dialogue as its safe fallback.
 */
export function useSession(): SessionState {
  const [step, setStep] = React.useState(0)
  const [lines, setLines] = React.useState<TranscriptLine[]>([
    scenarioScript.opening,
  ])
  const [rubric, setRubric] = React.useState<Record<string, RubricStatus>>(() =>
    Object.fromEntries(
      sessionCriteria.map((criterion) => [criterion.id, "unevaluated"])
    )
  )
  const [connection, setConnection] = React.useState<ConnectionState>("connected")
  const [speaking, setSpeaking] = React.useState(false)
  const [characterReplying, setCharacterReplying] = React.useState(false)
  const [elapsedSeconds, setElapsedSeconds] = React.useState(0)
  const [ended, setEnded] = React.useState(false)

  const speakingRef = React.useRef(false)
  const stepRef = React.useRef(0)
  const secondsRef = React.useRef(0)
  const replyingRef = React.useRef(false)
  const timers = React.useRef<number[]>([])
  const finishedRef = React.useRef(false)

  const finished = ended || step >= scenarioScript.turns.length

  React.useEffect(() => {
    finishedRef.current = finished
  }, [finished])

  React.useEffect(() => {
    if (finished) return
    const ticker = window.setInterval(() => {
      secondsRef.current += 1
      setElapsedSeconds(secondsRef.current)
    }, 1000)
    return () => window.clearInterval(ticker)
  }, [finished])

  React.useEffect(() => {
    const pending = timers.current
    return () => pending.forEach((id) => window.clearTimeout(id))
  }, [])

  function schedule(task: () => void, delay: number) {
    timers.current.push(window.setTimeout(task, delay))
  }

  const endSession = React.useCallback(() => {
    speakingRef.current = false
    setSpeaking(false)
    setEnded(true)
  }, [])

  const startSpeaking = React.useCallback(() => {
    if (replyingRef.current) return
    if (finishedRef.current) return
    if (stepRef.current >= scenarioScript.turns.length) return
    if (speakingRef.current) return
    speakingRef.current = true
    setSpeaking(true)
  }, [])

  const stopSpeaking = React.useCallback(() => {
    if (!speakingRef.current) return
    speakingRef.current = false
    setSpeaking(false)

    const turn = scenarioScript.turns[stepRef.current]
    if (!turn) return
    stepRef.current += 1
    setStep(stepRef.current)

    setLines((prev) => [
      ...prev,
      { ...turn.employee, time: formatTime(secondsRef.current) },
    ])

    if (turn.rubricUpdate) {
      setRubric((prev) => ({ ...prev, ...turn.rubricUpdate }))
    }

    // Scripted connection drop, so all three states show up during a session.
    if (turn.connectionDrop) {
      setConnection("disconnected")
      schedule(() => setConnection("reconnecting"), 900)
      schedule(() => setConnection("connected"), 2600)
    }

    const character = turn.character
    if (character) {
      replyingRef.current = true
      setCharacterReplying(true)
      schedule(() => {
        setLines((prev) => [
          ...prev,
          { ...character, time: formatTime(secondsRef.current) },
        ])
        replyingRef.current = false
        setCharacterReplying(false)
      }, 1400)
    }
  }, [])

  return {
    scenario: scenarioInfo,
    lines,
    rubric,
    connection,
    speaking,
    characterReplying,
    elapsedSeconds,
    finished,
    startSpeaking,
    stopSpeaking,
    endSession,
  }
}
