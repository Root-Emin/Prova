import { cn } from "@/lib/utils"

/**
 * Evidence quote pulled from the transcript. Visually distinct from system
 * copy: 3px brand-green rule on the left, sage surface, mono font.
 */
function TranscriptQuote({
  speaker,
  timestamp,
  children,
  className,
}: {
  speaker: string
  timestamp?: string
  children: React.ReactNode
  className?: string
}) {
  return (
    <figure className={cn("prova-quote rounded-r-md px-3 py-2", className)}>
      <figcaption className="prova-meta flex items-center justify-between font-sans uppercase">
        <span>{speaker}</span>
        {timestamp ? <span>{timestamp}</span> : null}
      </figcaption>
      <blockquote className="mt-1 text-sm leading-relaxed">{children}</blockquote>
    </figure>
  )
}

export { TranscriptQuote }
