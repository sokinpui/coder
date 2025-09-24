import {
  IconButton,
  Tooltip,
  Box,
} from '@mui/material';
import {
  Replay as ReplayIcon,
  PlaylistAddCheck as PlaylistAddCheckIcon,
  Edit as EditIcon,
  CallSplit as CallSplitIcon,
  Delete as DeleteIcon,
} from '@mui/icons-material';
import { CopyButton } from '../CopyButton';

interface MessageActionsProps {
  isUser: boolean;
  isAI: boolean;
  isImage: boolean;
  isGenerating: boolean;
  isEditing: boolean;
  isFloatingChat: boolean;
  messageContent: string;
  onEditStart: () => void;
  onApplyItf: () => void;
  onBranchFrom: () => void;
  onRegenerate: () => void;
  onDelete: () => void;
}

export function MessageActions({
  isUser,
  isAI,
  isImage,
  isGenerating,
  isEditing,
  isFloatingChat,
  messageContent,
  onEditStart,
  onApplyItf,
  onBranchFrom,
  onRegenerate,
  onDelete,
}: MessageActionsProps) {
  if (isFloatingChat) {
    return (
      <Box sx={{ display: 'flex', alignItems: 'center' }}>
        {!isImage && <CopyButton content={messageContent} />}
      </Box>
    );
  }

  return (
    <Box sx={{ display: 'flex', alignItems: 'center' }}>
      {isUser && !isGenerating && !isEditing && (
        <Tooltip title="Edit" placement="left" enterDelay={1000}>
          <IconButton
            onClick={onEditStart}
            size="small"
            color="inherit"
            sx={{
              mr: 0.5,
              backgroundColor: (theme) => theme.palette.action.hover,
              '&:hover': {
                backgroundColor: (theme) =>
                  theme.palette.action.selected,
              },
            }}
          >
            <EditIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      )}
      {isAI && !isGenerating && (
        <Tooltip title="Apply" placement="left" enterDelay={1000}>
          <IconButton
            onClick={onApplyItf}
            size="small"
            color="inherit"
            sx={{
              mr: 0.5,
              backgroundColor: (theme) => theme.palette.action.hover,
              '&:hover': {
                backgroundColor: (theme) =>
                  theme.palette.action.selected,
              },
            }}
          >
            <PlaylistAddCheckIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      )}
      {(isUser || isAI) && !isGenerating && (
        <Tooltip title="Branch from here" placement="left" enterDelay={1000}>
          <IconButton
            onClick={onBranchFrom}
            size="small"
            color="inherit"
            sx={{
              mr: 0.5,
              backgroundColor: (theme) => theme.palette.action.hover,
              '&:hover': {
                backgroundColor: (theme) =>
                  theme.palette.action.selected,
              },
            }}
          >
            <CallSplitIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      )}
      {(isUser || isAI) && !isGenerating && (
        <Tooltip title="Regenerate" placement="left" enterDelay={1000}>
          <IconButton
            onClick={onRegenerate}
            size="small"
            color="inherit"
            sx={{
              mr: 0.5,
              backgroundColor: (theme) =>
                theme.palette.action.hover,
              '&:hover': {
                backgroundColor: (theme) =>
                  theme.palette.action.selected,
              },
            }}
          >
            <ReplayIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      )}
      {!isImage && <CopyButton content={messageContent} />}
      {!isGenerating && (
        <Tooltip title="Delete" placement="left" enterDelay={1000}>
          <IconButton
            onClick={onDelete}
            size="small"
            color="inherit"
            sx={{
              ml: 0.5,
              backgroundColor: (theme) => theme.palette.action.hover,
              '&:hover': {
                backgroundColor: (theme) =>
                  theme.palette.action.selected,
              },
            }}
          >
            <DeleteIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      )}
    </Box>
  );
}
