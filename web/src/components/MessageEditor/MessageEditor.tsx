import { useState, useContext } from 'react';
import { Box, IconButton, useTheme } from '@mui/material';
import { Check as CheckIcon, Close as CloseIcon } from '@mui/icons-material';
import CodeMirror from '@uiw/react-codemirror';
import { EditorView } from '@codemirror/view';
import { oneDark } from '@codemirror/theme-one-dark';
import { githubLight } from '@uiw/codemirror-theme-github';
import { markdown, markdownLanguage } from '@codemirror/lang-markdown';
import { languages } from '@codemirror/language-data';
import { AppContext } from '../../AppContext';

interface MessageEditorProps {
  initialContent: string;
  onSave: (newContent: string) => void;
  onCancel: () => void;
}

export function MessageEditor({ initialContent, onSave, onCancel }: MessageEditorProps) {
  const [content, setContent] = useState(initialContent);

  const theme = useTheme();
  const { codeTheme } = useContext(AppContext);

  const handleSave = () => {
    onSave(content);
  };

  const customBgTheme = EditorView.theme({
    "&": {
      backgroundColor: theme.palette.background.paper,
    },
    ".cm-gutters": {
      backgroundColor: theme.palette.background.paper,
    },
  });

  return (
    <Box sx={{ pt: 1 }}>
      <CodeMirror
        value={content}
        onChange={(val) => setContent(val)}
        height="auto"
        minHeight="100px"
        maxHeight="400px"
        theme={codeTheme === 'dark' ? oneDark : githubLight}
        extensions={[
          markdown({ base: markdownLanguage, codeLanguages: languages }),
          EditorView.lineWrapping,
          customBgTheme,
        ]}
        basicSetup={{
          lineNumbers: false,
          foldGutter: false,
          autocompletion: false,
          highlightActiveLine: false,
          highlightActiveLineGutter: false,
        }}
        autoFocus
      />
      <Box sx={{ display: 'flex', justifyContent: 'flex-end', mt: 1, gap: 1 }}>
        <IconButton onClick={onCancel} size="small">
          <CloseIcon />
        </IconButton>
        <IconButton onClick={handleSave} size="small" color="primary">
          <CheckIcon />
        </IconButton>
      </Box>
    </Box>
  );
}
