import { useContext } from 'react';
import { Box, Typography, useTheme } from '@mui/material'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneDark, oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { CopyButton } from '../CopyButton';
import { AppContext } from '../../AppContext';

interface CodeBlockProps {
  language: string;
  children: React.ReactNode;
}

export function CodeBlock({ language, children }: CodeBlockProps) {
  const theme = useTheme();
  const { codeTheme } = useContext(AppContext);
  const codeString = String(children).replace(/\n$/, '');
  const syntaxTheme = codeTheme === 'dark' ? oneDark : oneLight;

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
        }}
      >
        <Typography variant="caption" sx={{ color: theme.palette.text.secondary, textTransform: 'lowercase' }}>
          {language}
        </Typography>
        <CopyButton content={codeString} />
      </Box>
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
    </Box>
  );
}
