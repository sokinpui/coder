import { memo, useState, useEffect } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import {
  Box,
  Paper,
  Typography,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  Replay as ReplayIcon,
  PlaylistAddCheck as PlaylistAddCheckIcon,
  Edit as EditIcon,
  CallSplit as CallSplitIcon,
  Delete as DeleteIcon,
} from '@mui/icons-material';
import type { Message } from '../../types';
import { CopyButton } from '../CopyButton';
import { CodeBlock } from '../CodeBlock';
import { MessageEditor } from '../MessageEditor';

interface MessageItemProps {
  message: Message;
  index: number;
  isGenerating: boolean;
  onRegenerate: (index: number) => void;
  onApplyItf: (content: string) => void;
  onEditMessage: (index: number, content: string) => void;
  onBranchFrom: (index: number) => void;
  onDeleteMessage: (index: number) => void;
  isFloatingChat?: boolean;
}

function MessageItemComponent({
  message,
  index,
  isGenerating,
  onRegenerate,
  onApplyItf,
  onEditMessage,
  onBranchFrom,
  onDeleteMessage,
  isFloatingChat = false,
}: MessageItemProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editedContent, setEditedContent] = useState(message.content);

  // Update editedContent if the original message content changes (e.g., after an external edit or load)
  useEffect(() => {
    setEditedContent(message.content);
  }, [message.content]);

  const handleEditStart = () => {
    setIsEditing(true);
  };

  const handleEditSave = (newContent: string) => {
    onEditMessage(index, newContent);
    setEditedContent(newContent); // Update local state after save
    setIsEditing(false);
  };

  const handleEditCancel = () => {
    setEditedContent(message.content); // Revert to original content on cancel
    setIsEditing(false);
  };

  const handleRegenerateClick = () => {
    if (isEditing) {
      onEditMessage(index, editedContent); // Save the current edited content before regenerating
      setIsEditing(false); // Exit editing mode
    }
    onRegenerate(index); // Then trigger regeneration
  };

  const isUser = message.sender === 'User';
  const isError = message.sender === 'Error';
  const isAI = message.sender === 'AI';
  const isImage = message.sender === 'Image';

  if (isImage) {
    return (
      <Paper
        elevation={1}
        sx={{
          p: 1,
          mb: 1.5,
          maxWidth: '50%',
          alignSelf: 'flex-end', // Images are from user, so align right
          bgcolor: 'background.paper',
        }}
      >
        <img
          src={`/files/${message.content}`}
          alt={message.content}
          style={{ maxWidth: '100%', height: 'auto', borderRadius: '8px' }}
        />
        <Typography variant="caption" sx={{ display: 'block', color: 'text.secondary', mt: 0.5 }}>
          {message.content.split('/').pop()}
        </Typography>
      </Paper>
    );
  }

  return (
    <Paper
      elevation={1}
      sx={{
        position: 'relative',
        mb: 1.5,
        maxWidth: '100%',
        width: isEditing || !isUser ? '100%' : 'auto',
        alignSelf: isUser ? 'flex-end' : 'flex-start',
        bgcolor: 'background.paper',
        color: isError ? 'error.main' : 'text.primary',
        borderTopLeftRadius: !isUser ? 0 : undefined,
        borderTopRightRadius: isUser ? 0 : undefined,
      }}
    >
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          position: 'sticky',
          top: 0,
          zIndex: 1,
          bgcolor: 'background.default',
          py: 0.5,
          px: 1.5,
          borderTopLeftRadius: (theme) =>
            !isUser ? 0 : theme.shape.borderRadius,
          borderTopRightRadius: (theme) =>
            isUser ? 0 : theme.shape.borderRadius,
        }}
      >
        <Typography variant="subtitle2" sx={{ fontWeight: 'bold' }}>
          {message.sender}
        </Typography>
        <Box sx={{ display: 'flex', alignItems: 'center' }}>
          {isUser && !isGenerating && !isEditing && (
            <Tooltip title="Edit" placement="left" enterDelay={1000}>
              <IconButton
                onClick={handleEditStart}
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
          {isAI && !isGenerating && !isFloatingChat && (
            <Tooltip title="Apply" placement="left" enterDelay={1000}>
              <IconButton
                onClick={() => onApplyItf(message.content)}
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
          {(isUser || isAI) && !isGenerating && !isFloatingChat && (
            <Tooltip title="Branch from here" placement="left" enterDelay={1000}>
              <IconButton
                onClick={() => onBranchFrom(index)}
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
          {(isUser || isAI) && !isGenerating && !isFloatingChat && (
            <Tooltip title="Regenerate" placement="left" enterDelay={1000}>
              <IconButton // Modified: Use handleRegenerateClick
                onClick={handleRegenerateClick}
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
          <CopyButton content={message.content} />
          {!isGenerating && !isFloatingChat && (
            <Tooltip title="Delete" placement="left" enterDelay={1000}>
              <IconButton
                onClick={() => onDeleteMessage(index)}
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
      </Box>
      <Box
        className="message-content"
        sx={{
          '& pre': {
            whiteSpace: 'pre-wrap',
            wordWrap: 'break-word',
            fontFamily: 'monospace',
          },
          '& code': {
            fontFamily: 'monospace',
            backgroundColor: 'action.hover',
            border: (theme) => `1px solid ${theme.palette.divider}`,
            borderRadius: (theme) => `${theme.shape.borderRadius / 3}px`,
            px: '4px',
            py: '2px',
          },
          '& pre > code': {
            display: 'block',
            p: 1,
            backgroundColor: 'action.hover',
            border: (theme) => `1px solid ${theme.palette.divider}`,
            borderRadius: (theme) => `${theme.shape.borderRadius}px`,
          },
          '& table': {
            borderCollapse: 'collapse',
            my: 1,
            '& th, & td': {
              border: (theme) => `1px solid ${theme.palette.divider}`,
              p: 1,
            },
            '& th': {
              fontWeight: 'bold',
              textAlign: 'left',
            },
            '& thead': {
              backgroundColor: 'action.hover',
            },
          },
          px: 1.5,
          pb: 1.5,
        }}
      >
        {isEditing ? (
          <MessageEditor
            initialContent={editedContent} // Pass editedContent
            onSave={handleEditSave}
            onCancel={handleEditCancel}
            onChange={setEditedContent} // New prop for MessageEditor to update editedContent
          />
        ) : (
          <>
            {message.sender === 'AI' || message.sender === 'User' ? (
              <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                components={{
                  code({ className, children, ...props }) {
                    const match = /language-(\w+)/.exec(className || '');
                    if (match) {
                      return (
                        <CodeBlock language={match[1]}>{children}</CodeBlock>
                      );
                    }
                    return (
                      <code className={className} {...props}>
                        {children}
                      </code>
                    );
                  },
                }}
              >
                {message.content}
              </ReactMarkdown>
            ) : (
              <Typography component="pre">{message.content}</Typography>
            )}
          </>
        )}
      </Box>
    </Paper>
  );
}

export const MessageItem = memo(MessageItemComponent);
