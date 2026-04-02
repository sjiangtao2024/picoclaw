import { normalizeUnixTimestamp } from "@/features/chat/state"
import type { ChatAttachment } from "@/store/chat"
import { updateChatStore } from "@/store/chat"

export interface PicoMessage {
  type: string
  id?: string
  session_id?: string
  timestamp?: number | string
  payload?: Record<string, unknown>
}

export function handlePicoMessage(
  message: PicoMessage,
  expectedSessionId: string,
) {
  if (message.session_id && message.session_id !== expectedSessionId) {
    return
  }

  const payload = message.payload || {}

  switch (message.type) {
    case "message.create": {
      const content = (payload.content as string) || ""
      const messageId = (payload.message_id as string) || `pico-${Date.now()}`
      const timestamp =
        message.timestamp !== undefined &&
        Number.isFinite(Number(message.timestamp))
          ? normalizeUnixTimestamp(Number(message.timestamp))
          : Date.now()

      updateChatStore((prev) => ({
        messages: [
          ...prev.messages,
          {
            id: messageId,
            role: "assistant",
            content,
            timestamp,
          },
        ],
        isTyping: false,
      }))
      break
    }

    case "message.update": {
      const content = (payload.content as string) || ""
      const messageId = payload.message_id as string
      if (!messageId) {
        break
      }

      updateChatStore((prev) => ({
        messages: prev.messages.map((msg) =>
          msg.id === messageId ? { ...msg, content } : msg,
        ),
      }))
      break
    }

    case "media.create": {
      const attachment = toChatAttachment(payload)
      if (!attachment) {
        break
      }

      const messageId = (payload.message_id as string) || `pico-media-${Date.now()}`
      const timestamp =
        message.timestamp !== undefined &&
        Number.isFinite(Number(message.timestamp))
          ? normalizeUnixTimestamp(Number(message.timestamp))
          : Date.now()
      const caption = (payload.caption as string) || ""

      updateChatStore((prev) => ({
        messages: [
          ...prev.messages,
          {
            id: messageId,
            role: "assistant",
            content: caption,
            attachments: [attachment],
            timestamp,
          },
        ],
        isTyping: false,
      }))
      break
    }

    case "typing.start":
      updateChatStore({ isTyping: true })
      break

    case "typing.stop":
      updateChatStore({ isTyping: false })
      break

    case "error":
      console.error("Pico error:", payload)
      updateChatStore({ isTyping: false })
      break

    case "pong":
      break

    default:
      console.log("Unknown pico message type:", message.type)
  }
}

function toChatAttachment(
  payload: Record<string, unknown>,
): ChatAttachment | null {
  const dataUrl = payload.data_url as string
  if (!dataUrl) {
    return null
  }

  const mediaType = payload.type
  const type: ChatAttachment["type"] =
    mediaType === "image" ||
    mediaType === "audio" ||
    mediaType === "video" ||
    mediaType === "file"
      ? mediaType
      : "file"

  return {
    type,
    filename: (payload.filename as string) || undefined,
    contentType: (payload.content_type as string) || undefined,
    dataUrl,
    caption: (payload.caption as string) || undefined,
  }
}
