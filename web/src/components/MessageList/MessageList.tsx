import { useRef, useEffect, useCallback, memo } from "react";
import ReactMarkdown from "react-markdown";
import { Box, Typography, Paper } from "@mui/material";
import type { Message } from "../../types";
import { TypingIndicator } from "../TypingIndicator";
import { MessageItem } from "../MessageItem";

interface MessageListProps {
  messages: Message[];
  isGenerating: boolean;
  onRegenerate: (userMessageIndex: number) => void;
  onApplyItf: (content: string) => void;
  onEditMessage: (index: number, content: string) => void;
  onBranchFrom: (messageIndex: number) => void;
  onDeleteMessage: (index: number) => void;
  isFloatingChat?: boolean; // Already exists
}

function MessageListComponent({
  messages,
  isGenerating,
  onRegenerate,
  onApplyItf,
  onEditMessage,
  onBranchFrom,
  onDeleteMessage,
  isFloatingChat = false, // Already exists
}: MessageListProps) {
  const scrollContainerRef = useRef<HTMLDivElement | null>(null);
  const messagesEndRef = useRef<HTMLDivElement | null>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    // Only auto-scroll if the user is near the bottom.
    // This prevents the view from jumping down if they've scrolled up.
    const scrollThreshold = 100; // pixels
    const isScrolledToBottom =
      container.scrollHeight - container.scrollTop <=
      container.clientHeight + scrollThreshold;

    if (isScrolledToBottom) scrollToBottom();
  }, [messages]);

  const messagesRef = useRef(messages);
  useEffect(() => {
    messagesRef.current = messages;
  }, [messages]);

  const handleRegenerate = useCallback(
    (currentIndex: number) => {
      let userMessageIndex = -1;
      for (let i = currentIndex; i >= 0; i--) {
        if (messagesRef.current[i].sender === "User") {
          userMessageIndex = i;
          break;
        }
      }
      if (userMessageIndex !== -1) {
        onRegenerate(userMessageIndex);
      }
    },
    [onRegenerate],
  );

  return (
    <>
      <Box
        ref={scrollContainerRef}
        sx={{
          flexGrow: 1,
          overflowY: "auto",
          px: 2,
          pb: 2,
          display: "flex",
          flexDirection: "column",
        }}
      >
        {messages.map((msg, index) => {
          if (msg.sender === "System") {
            return (
              <Typography
                key={index}
                variant="caption"
                component="div"
                sx={{
                  alignSelf: "center",
                  fontStyle: "italic",
                  color: "text.secondary",
                  mb: 1.5,
                  "& blockquote": {
                    borderLeft: (theme) => `4px solid ${theme.palette.divider}`,
                    pl: 1.5,
                    m: 0,
                    fontStyle: "normal",
                  },
                }}
              >
                <ReactMarkdown>{msg.content}</ReactMarkdown>
              </Typography>
            );
          }

          return (
            <MessageItem
              key={index}
              message={msg}
              index={index}
              isGenerating={isGenerating}
              onRegenerate={handleRegenerate}
              onApplyItf={onApplyItf}
              onEditMessage={onEditMessage}
              onBranchFrom={onBranchFrom}
              onDeleteMessage={onDeleteMessage}
              isFloatingChat={isFloatingChat}
            />
          );
        })}
        {isGenerating &&
          messages.length > 0 &&
          messages[messages.length - 1].sender === "User" && (
            <Paper
              elevation={1}
              sx={{
                p: 1.5,
                mb: 1.5,
                maxWidth: "80%",
                alignSelf: "flex-start",
                bgcolor: "background.paper",
                color: "text.primary",
              }}
            >
              <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                <Typography variant="body2">Thinking</Typography>
                <TypingIndicator />
              </Box>
            </Paper>
          )}
        <div ref={messagesEndRef} />
      </Box>
    </>
  );
}

export const MessageList = memo(MessageListComponent);
