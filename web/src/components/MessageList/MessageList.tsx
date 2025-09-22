import { useState, useRef, useEffect, useCallback, memo } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import {
  Box,
  Typography,
  CircularProgress,
  Paper,
  Popper,
  Fade,
} from "@mui/material";
import type { Message } from "../../types";
import { HighlightMenu } from "../HighlightMenu";
import { MessageItem } from "../MessageItem";

interface MessageListProps {
  messages: Message[];
  isGenerating: boolean;
  onRegenerate: (userMessageIndex: number) => void;
  onApplyItf: (content: string) => void;
  onEditMessage: (index: number, content: string) => void;
  onBranchFrom: (messageIndex: number) => void;
  onDeleteMessage: (index: number) => void;
  onAskAI: (text: string) => void;
}

function MessageListComponent({
  messages,
  isGenerating,
  onRegenerate,
  onApplyItf,
  onEditMessage,
  onBranchFrom,
  onDeleteMessage,
  onAskAI,
}: MessageListProps) {
  const scrollContainerRef = useRef<HTMLDivElement | null>(null);
  const messagesEndRef = useRef<HTMLDivElement | null>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const [highlightMenuState, setHighlightMenuState] = useState<{
    open: boolean;
    anchorEl: { getBoundingClientRect: () => DOMRect } | null;
    selectedText: string;
  }>({
    open: false,
    anchorEl: null,
    selectedText: "",
  });

  const findUserMessageIndex = (currentIndex: number) => {
    let userMessageIndex = -1;
    for (let i = currentIndex; i >= 0; i--) {
      if (messages[i].sender === "User") {
        userMessageIndex = i;
        break;
      }
    }
    return userMessageIndex;
  };

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

  const handleCloseHighlightMenu = useCallback(() => {
    setHighlightMenuState((prev) => ({ ...prev, open: false }));
  }, []);

  const handleMouseUp = (event: React.MouseEvent) => {
    if (menuRef.current && menuRef.current.contains(event.target as Node)) {
      return;
    }

    setTimeout(() => {
      const selection = window.getSelection();
      if (selection && selection.toString().trim().length > 0) {
        const range = selection.getRangeAt(0);

        const virtualEl = {
          getBoundingClientRect: () => range.getBoundingClientRect(),
        };

        setHighlightMenuState({
          open: true,
          anchorEl: virtualEl,
          selectedText: selection.toString(),
        });
      } else {
        handleCloseHighlightMenu();
      }
    }, 10);
  };

  const handleCopySuccess = () => {
    setTimeout(() => {
      handleCloseHighlightMenu();
      window.getSelection()?.removeAllRanges();
    }, 500);
  };

  const handleAskAI = (text: string) => {
    onAskAI(text);
    handleCloseHighlightMenu();
  };

  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const handleScroll = () => {
      if (highlightMenuState.open) handleCloseHighlightMenu();
    };

    container.addEventListener("scroll", handleScroll, { passive: true });
    return () => container.removeEventListener("scroll", handleScroll);
  }, [highlightMenuState.open, handleCloseHighlightMenu]);

  return (
    <>
      <Box
        ref={scrollContainerRef}
        onMouseUp={handleMouseUp}
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
                sx={{
                  alignSelf: "center",
                  fontStyle: "italic",
                  color: "text.secondary",
                  mb: 1.5,
                }}
              >
                {msg.content}
              </Typography>
            );
          }

          return (
            <MessageItem
              key={index}
              message={msg}
              index={index}
              isGenerating={isGenerating}
              onRegenerate={() => {
                const userMsgIndex = findUserMessageIndex(index);
                if (userMsgIndex !== -1) onRegenerate(userMsgIndex);
              }}
              onApplyItf={onApplyItf}
              onEditMessage={onEditMessage}
              onBranchFrom={onBranchFrom}
              onDeleteMessage={onDeleteMessage}
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
              <Box sx={{ display: "flex", alignItems: "center" }}>
                <CircularProgress size={20} sx={{ mr: 1.5 }} />
                <Typography variant="body2">AI is thinking...</Typography>
              </Box>
            </Paper>
          )}
        <div ref={messagesEndRef} />
      </Box>
      <Popper
        open={highlightMenuState.open}
        anchorEl={highlightMenuState.anchorEl}
        placement="top"
        transition
        sx={{ zIndex: 1300 }}
      >
        {({ TransitionProps }) => (
          <Fade {...TransitionProps} timeout={150}>
            <div ref={menuRef}>
              <HighlightMenu
                selectedText={highlightMenuState.selectedText}
                onCopySuccess={handleCopySuccess}
                onAskAI={handleAskAI}
              />
            </div>
          </Fade>
        )}
      </Popper>
    </>
  );
}

export const MessageList = memo(MessageListComponent);
