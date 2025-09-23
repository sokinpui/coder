import { useState, useRef, useEffect, useCallback } from "react";
import Draggable, {
  type DraggableData,
  type DraggableEvent,
} from "react-draggable";
import {
  Dialog,
  DialogTitle,
  DialogContent,
  Typography,
  IconButton,
  Box,
  Paper,
  type PaperProps,
  type Theme,
} from "@mui/material";
import CloseIcon from "@mui/icons-material/Close";
import MinimizeIcon from "@mui/icons-material/Minimize";
import WebAssetIcon from "@mui/icons-material/WebAsset";
import { MessageList } from "../MessageList";
import { ChatInput } from "../ChatInput";
import type { Message } from "../../types";

interface FloatingChatWindowProps {
  open: boolean;
  onClose: () => void;
  context: string;
  askAI: (params: {
    context: string;
    question: string;
    history: Message[];
    onChunk: (chunk: string) => void;
    onEnd: () => void;
    onError: (error: string) => void;
  }) => void;
}

export function FloatingChatWindow({
  open,
  onClose,
  context,
  askAI,
}: FloatingChatWindowProps) {
  const [isMinimized, setIsMinimized] = useState(false);
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const [messages, setMessages] = useState<Message[]>([]);
  const [isGenerating, setIsGenerating] = useState(false);

  useEffect(() => {
    if (open) {
      setIsMinimized(false);
      setMessages([
        {
          sender: "System",
          content: `Asking about the following snippet:\n\n---\n${context}\n---\n\nWhat is your question?`,
        },
      ]);
    } else {
      // Reset state on close
      setMessages([]);
      setIsGenerating(false);
    }
  }, [open, context]);

  const handleMinimizeToggle = () => {
    setIsMinimized(!isMinimized);
  };

  const handleDrag = (_e: DraggableEvent, data: DraggableData) => {
    setPosition({ x: data.x, y: data.y });
  };

  const handleSendMessage = useCallback(
    (question: string) => {
      if (!question.trim()) return;

      setIsGenerating(true);
      setMessages((prev) => [...prev, { sender: "User", content: question }]);

      // The history sent to backend should not include the system message
      const chatHistory = messages.filter((m) => m.sender !== "System");

      askAI({
        context: context,
        question: question,
        history: chatHistory,
        onChunk: (chunk) => {
          setMessages((prev) => {
            const last = prev[prev.length - 1];
            if (last?.sender === "AI") {
              const newMessages = [...prev];
              newMessages[newMessages.length - 1] = {
                ...last,
                content: last.content + chunk,
              };
              return newMessages;
            }
            return [...prev, { sender: "AI", content: chunk }];
          });
        },
        onEnd: () => {
          setIsGenerating(false);
        },
        onError: (error) => {
          setMessages((prev) => [
            ...prev,
            { sender: "Error", content: error },
          ]);
          setIsGenerating(false);
        },
      });
    },
    [askAI, context, messages],
  );

  // Dummy functions for MessageList props that are not applicable here
  const noOp = () => {};

  return (
    <Dialog
      open={open}
      onClose={onClose}
      PaperComponent={(props: PaperProps) => {
        const nodeRef = useRef(null);
        return (
          <Draggable
            nodeRef={nodeRef}
            handle="#draggable-dialog-title"
            cancel={'[class*="MuiDialogContent-root"], .MuiButtonBase-root'}
            position={position}
            onDrag={handleDrag}
          >
            <Paper ref={nodeRef} {...props} />
          </Draggable>
        );
      }}
      PaperProps={
        {
          sx: {
            position: "fixed",
            bottom: 20,
            right: 20,
            m: 0,
            maxHeight: "80vh",
            borderRadius: 2,
            overflow: "hidden",
            width: isMinimized ? 250 : 600,
            height: isMinimized ? "auto" : 700,
            transition: (theme: Theme) =>
              theme.transitions.create(["width", "height"]),
            display: "flex",
            flexDirection: "column",
            resize: "none",
          },
        }
      }
      BackdropProps={{
        style: {
          backgroundColor: "transparent",
        },
      }}
      hideBackdrop
    >
      <DialogTitle
        sx={{
          p: 1.5,
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          cursor: "move",
        }}
        id="draggable-dialog-title"
      >
        <Typography variant="h6" component="div" sx={{ fontSize: "1rem" }}>
          Ask AI
        </Typography>
        <Box>
          <IconButton
            onClick={handleMinimizeToggle}
            size="small"
            sx={{ mr: 0.5 }}
          >
            {isMinimized ? (
              <WebAssetIcon fontSize="small" />
            ) : (
              <MinimizeIcon fontSize="small" />
            )}
          </IconButton>
          <IconButton onClick={onClose} size="small">
            <CloseIcon fontSize="small" />
          </IconButton>
        </Box>
      </DialogTitle>
      {!isMinimized && (
        <DialogContent
          dividers
          sx={{
            p: 0,
            flexGrow: 1,
            display: "flex",
            flexDirection: "column",
            overflow: "hidden",
          }}
        >
          <MessageList
            messages={messages}
            isGenerating={isGenerating}
            onRegenerate={noOp}
            onApplyItf={noOp}
            onEditMessage={noOp}
            onBranchFrom={noOp}
            onDeleteMessage={noOp}
            isFloatingChat={true} // Keep this to hide other buttons
            onAskAI={undefined} // Explicitly disable Ask AI in floating chat
          />
          <ChatInput
            sendMessage={handleSendMessage}
            cancelGeneration={noOp} // Cancellation not implemented for askAI yet
            isGenerating={isGenerating}
          />
        </DialogContent>
      )}
    </Dialog>
  );
}
