import { useState, useRef } from "react";
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
} from "@mui/material";
import CloseIcon from "@mui/icons-material/Close";
import MinimizeIcon from "@mui/icons-material/Minimize";
import WebAssetIcon from "@mui/icons-material/WebAsset";

interface FloatingChatWindowProps {
  open: boolean;
  onClose: () => void;
  context: string;
}

interface DraggablePaperProps extends PaperProps {
  position: { x: number; y: number };
  onDrag: (e: DraggableEvent, data: DraggableData) => void;
}

function DraggablePaper(props: DraggablePaperProps) {
  const { position, onDrag, ...paperProps } = props;
  const nodeRef = useRef(null);

  return (
    <Draggable
      nodeRef={nodeRef}
      handle="#draggable-dialog-title"
      cancel={'[class*="MuiDialogContent-root"]'}
      position={position}
      onDrag={onDrag}
    >
      <Paper ref={nodeRef} {...paperProps} />
    </Draggable>
  );
}

export function FloatingChatWindow({
  open,
  onClose,
  context,
}: FloatingChatWindowProps) {
  const [isMinimized, setIsMinimized] = useState(false);
  const [position, setPosition] = useState({ x: 0, y: 0 });

  const handleMinimizeToggle = () => {
    setIsMinimized(!isMinimized);
  };

  const handleDrag = (_e: DraggableEvent, data: DraggableData) => {
    setPosition({ x: data.x, y: data.y });
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      PaperComponent={DraggablePaper}
      PaperProps={
        {
          position,
          onDrag: handleDrag,
          sx: {
            position: "fixed",
            bottom: 20,
            right: 20,
            m: 0,
            maxHeight: "80vh",
            borderRadius: 2,
            overflow: "hidden",
            width: isMinimized ? 250 : 400,
            height: isMinimized ? "auto" : 500,
            transition: (theme) =>
              theme.transitions.create(["width", "height"]),
            display: "flex",
            flexDirection: "column",
          },
        } as any
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
          sx={{ p: 1.5, flexGrow: 1, display: "flex", flexDirection: "column" }}
        >
          <Box
            sx={{
              p: 1,
              bgcolor: "action.hover",
              borderRadius: 1,
              whiteSpace: "pre-wrap",
              wordBreak: "break-word",
              fontFamily: "monospace",
              fontSize: "0.875rem",
              flexGrow: 1,
              overflowY: "auto",
            }}
          >
            {context}
          </Box>
        </DialogContent>
      )}
    </Dialog>
  );
}
