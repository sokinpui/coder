import { useState, useRef, useCallback } from "react";
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
  useTheme,
} from "@mui/material";
import { alpha } from "@mui/material/styles";
import CloseIcon from "@mui/icons-material/Close";
import MinimizeIcon from "@mui/icons-material/Minimize";
import WebAssetIcon from "@mui/icons-material/WebAsset";

interface FloatingChatWindowProps {
  open: boolean;
  onClose: () => void;
  context: string;
}

interface ResizablePaperProps extends PaperProps {
  isMinimized: boolean;
  position: { x: number; y: number };
  size: { width: number | string; height: number | string };
  onDrag: (e: DraggableEvent, data: DraggableData) => void;
  onMouseDownResize: (e: React.MouseEvent, direction: string) => void;
}

function ResizablePaper(props: ResizablePaperProps) {
  const {
    isMinimized,
    position,
    size,
    onDrag,
    onMouseDownResize,
    ...paperProps
  } = props;
  const theme = useTheme();
  const nodeRef = useRef(null);

  const resizerBaseStyle: React.CSSProperties = {
    position: "absolute",
    zIndex: 10,
  };

  const resizerSx = {
    backgroundColor: alpha(theme.palette.primary.main, 0.5),
  };

  const resizers = !isMinimized && (
    <>
      <Box
        className="resizer"
        onMouseDown={(e) => onMouseDownResize(e, "top")}
        sx={{
          ...resizerSx,
          ...resizerBaseStyle,
          top: -3,
          left: 0,
          width: "100%",
          height: 6,
          cursor: "n-resize",
        }}
      />
      <Box
        className="resizer"
        onMouseDown={(e) => onMouseDownResize(e, "right")}
        sx={{
          ...resizerSx,
          ...resizerBaseStyle,
          top: 0,
          right: -3,
          width: 6,
          height: "100%",
          cursor: "e-resize",
        }}
      />
      <Box
        className="resizer"
        onMouseDown={(e) => onMouseDownResize(e, "bottom")}
        sx={{
          ...resizerSx,
          ...resizerBaseStyle,
          bottom: -3,
          left: 0,
          width: "100%",
          height: 6,
          cursor: "s-resize",
        }}
      />
      <Box
        className="resizer"
        onMouseDown={(e) => onMouseDownResize(e, "left")}
        sx={{
          ...resizerSx,
          ...resizerBaseStyle,
          top: 0,
          left: -3,
          width: 6,
          height: "100%",
          cursor: "w-resize",
        }}
      />
      <Box
        className="resizer"
        onMouseDown={(e) => onMouseDownResize(e, "top-left")}
        sx={{
          ...resizerSx,
          ...resizerBaseStyle,
          top: -3,
          left: -3,
          width: 10,
          height: 10,
          cursor: "nw-resize",
        }}
      />
      <Box
        className="resizer"
        onMouseDown={(e) => onMouseDownResize(e, "top-right")}
        sx={{
          ...resizerSx,
          ...resizerBaseStyle,
          top: -3,
          right: -3,
          width: 10,
          height: 10,
          cursor: "ne-resize",
        }}
      />
      <Box
        className="resizer"
        onMouseDown={(e) => onMouseDownResize(e, "bottom-left")}
        sx={{
          ...resizerSx,
          ...resizerBaseStyle,
          bottom: -3,
          left: -3,
          width: 10,
          height: 10,
          cursor: "sw-resize",
        }}
      />
      <Box
        className="resizer"
        onMouseDown={(e) => onMouseDownResize(e, "bottom-right")}
        sx={{
          ...resizerSx,
          ...resizerBaseStyle,
          bottom: -3,
          right: -3,
          width: 10,
          height: 10,
          cursor: "se-resize",
        }}
      />
    </>
  );

  return (
    <Draggable
      nodeRef={nodeRef}
      handle="#draggable-dialog-title"
      cancel={'[class*="MuiDialogContent-root"], .resizer'}
      position={position}
      onDrag={onDrag}
    >
      <Paper
        ref={nodeRef}
        {...paperProps}
        style={{ ...paperProps.style, width: size.width, height: size.height }}
      >
        {paperProps.children}
        {resizers}
      </Paper>
    </Draggable>
  );
}

export function FloatingChatWindow({
  open,
  onClose,
  context,
}: FloatingChatWindowProps) {
  const [isMinimized, setIsMinimized] = useState(false);
  const [size, setSize] = useState({ width: 400, height: 500 });
  const [position, setPosition] = useState({ x: 0, y: 0 });

  const handleMinimizeToggle = () => {
    setIsMinimized(!isMinimized);
  };

  const handleDrag = (e: DraggableEvent, data: DraggableData) => {
    setPosition({ x: data.x, y: data.y });
  };

  const handleMouseDownResize = useCallback(
    (e: React.MouseEvent, direction: string) => {
      e.preventDefault();
      e.stopPropagation();
      const startX = e.clientX;
      const startY = e.clientY;
      const startWidth = size.width;
      const startHeight = size.height;
      const startPos = { ...position };

      const handleMouseMove = (e: MouseEvent) => {
        const dx = e.clientX - startX;
        const dy = e.clientY - startY;

        let newWidth = startWidth;
        let newHeight = startHeight;
        let newX = startPos.x;
        let newY = startPos.y;

        if (direction.includes("right")) newWidth = startWidth + dx;
        if (direction.includes("left")) {
          newWidth = startWidth - dx;
          newX = startPos.x + dx;
        }
        if (direction.includes("bottom")) newHeight = startHeight + dy;
        if (direction.includes("top")) {
          newHeight = startHeight - dy;
          newY = startPos.y + dy;
        }

        if (newWidth > 250) {
          setSize((s) => ({ ...s, width: newWidth }));
          if (direction.includes("left"))
            setPosition((p) => ({ ...p, x: newX }));
        }
        if (newHeight > 200) {
          setSize((s) => ({ ...s, height: newHeight }));
          if (direction.includes("top"))
            setPosition((p) => ({ ...p, y: newY }));
        }
      };

      const handleMouseUp = () => {
        document.removeEventListener("mousemove", handleMouseMove);
        document.removeEventListener("mouseup", handleMouseUp);
      };

      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    },
    [size.width, size.height, position],
  );

  return (
    <Dialog
      open={open}
      onClose={onClose}
      PaperComponent={ResizablePaper}
      PaperProps={
        {
          isMinimized,
          position,
          size: {
            width: isMinimized ? 250 : size.width,
            height: isMinimized ? "auto" : size.height,
          },
          onDrag: handleDrag,
          onMouseDownResize: handleMouseDownResize,
          sx: {
            position: "fixed",
            bottom: 20,
            right: 20,
            m: 0,
            maxHeight: "80vh",
            borderRadius: 2,
            overflow: "visible",
            transition: isMinimized
              ? (theme) => theme.transitions.create(["width", "height"])
              : "none",
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
