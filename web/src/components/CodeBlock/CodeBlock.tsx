import { useContext, useState } from 'react';
import { Box, Typography, useTheme, IconButton, Tooltip } from '@mui/material'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneDark, oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { CopyButton } from '../CopyButton';
import { AppContext } from '../../AppContext';
import { KeyboardArrowDown, KeyboardArrowUp } from '@mui/icons-material';

interface CodeBlockProps {
  language: string;
  children: React.ReactNode;
}

export function CodeBlock({ language, children }: CodeBlockProps) {
  const theme = useTheme();
  const { codeTheme } = useContext(AppContext);
  const codeString = String(children).replace(/\n$/, '');
  const syntaxTheme = codeTheme === 'dark' ? oneDark : oneLight;
  const [isCollapsed, setIsCollapsed] = useState(false);

  return (
    <Box sx={{ position: 'relative', my: 1, borderRadius: `${theme.shape.borderRadius}px`, overflow: 'hidden' }}>
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          backgroundColor: theme.palette.action.hover,
          px: 1.5,
          py: 0.5,
          cursor: 'pointer',
        }}
        onClick={() => setIsCollapsed(!isCollapsed)}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Tooltip title={isCollapsed ? 'Expand code' : 'Collapse code'} enterDelay={1000}>
            <IconButton size="small" sx={{ p: 0 }}>
              {isCollapsed ? <KeyboardArrowDown fontSize="small" /> : <KeyboardArrowUp fontSize="small" />}
            </IconButton>
          </Tooltip>
          <Typography variant="caption" sx={{ color: theme.palette.text.secondary, textTransform: 'lowercase' }}>
            {language}
          </Typography>
        </Box>
        <Box onClick={(e) => e.stopPropagation()}>
          <CopyButton content={codeString} />
        </Box>
      </Box>
      {!isCollapsed && (
        <SyntaxHighlighter
          style={syntaxTheme}
          language={language}
          customStyle={{
            margin: 0,
            padding: theme.spacing(1.5),
            whiteSpace: 'pre-wrap',
            fontSize: '0.95rem',
            overflowWrap: 'break-word',
          }}
        >
          {codeString}
        </SyntaxHighlighter>
      )}
    </Box>
  );
}
