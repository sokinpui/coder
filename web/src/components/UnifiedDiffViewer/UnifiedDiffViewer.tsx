import { Box, Typography, useTheme } from '@mui/material';
import { alpha } from '@mui/material/styles';

interface UnifiedDiffViewerProps {
  diff: string;
}

// A simple parser for unified diff format
function parseDiff(diff: string): { fileName: string; lines: string[] }[] {
  const files: { fileName: string; lines: string[] }[] = [];
  const diffLines = diff.split('\n');

  let currentFile: { fileName: string; lines: string[] } | null = null;

  for (const line of diffLines) {
    if (line.startsWith('diff --git')) {
      if (currentFile) {
        files.push(currentFile);
      }
      const parts = line.split(' ');
      const fileName = parts.length > 2 ? parts[2].substring(2) : 'unknown';
      currentFile = { fileName, lines: [] };
      continue;
    }

    if (currentFile) {
      currentFile.lines.push(line);
    }
  }

  if (currentFile) {
    files.push(currentFile);
  }

  return files;
}

export function UnifiedDiffViewer({ diff }: UnifiedDiffViewerProps) {
  const theme = useTheme();
  const fileDiffs = parseDiff(diff);

  if (fileDiffs.length === 0) {
    return <Typography sx={{ p: 2 }}>No changes to display.</Typography>;
  }

  const getLineStyle = (line: string) => {
    if (line.startsWith('+') && !line.startsWith('+++')) {
      return { backgroundColor: alpha(theme.palette.success.main, 0.15) };
    }
    if (line.startsWith('-') && !line.startsWith('---')) {
      return { backgroundColor: alpha(theme.palette.error.main, 0.15) };
    }
    if (line.startsWith('@@')) {
      return { color: theme.palette.text.secondary };
    }
    return {};
  };

  return (
    <Box sx={{ p: 1, overflow: 'auto', height: '100%', bgcolor: 'background.default' }}>
      {fileDiffs.map((file, fileIndex) => (
        <Box key={fileIndex} sx={{ mb: 2, border: 1, borderColor: 'divider', borderRadius: 1, overflow: 'hidden', fontFamily: 'monospace', fontSize: '0.875rem' }}>
          <Typography sx={{ p: 1, bgcolor: 'action.hover', fontWeight: 'bold', fontFamily: 'inherit' }}>
            {file.fileName}
          </Typography>
          <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
            {file.lines.map((line, index) => (
              <Box key={index} sx={{ ...getLineStyle(line), display: 'flex', px: 1 }}>
                <Box component="span" sx={{ minWidth: '1em', userSelect: 'none', pr: 1 }}>
                  {line.startsWith('+') && !line.startsWith('+++') ? '+' : line.startsWith('-') && !line.startsWith('---') ? '-' : ' '}
                </Box>
                <Typography component="span" sx={{ fontFamily: 'inherit', fontSize: 'inherit' }}>
                  {line.substring(1)}
                </Typography>
              </Box>
            ))}
          </pre>
        </Box>
      ))}
    </Box>
  );
}
