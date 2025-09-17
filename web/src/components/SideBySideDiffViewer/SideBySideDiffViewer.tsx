import { Box, Typography, useTheme } from '@mui/material';
import { alpha } from '@mui/material/styles';

interface LinePair {
  left?: { content: string; type: 'remove' | 'context'; lineNumber: number };
  right?: { content: string; type: 'add' | 'context'; lineNumber: number };
}

interface Hunk {
  header: string;
  linePairs: LinePair[];
}

interface FileDiff {
  fileName: string;
  hunks: Hunk[];
}

// A simple parser for unified diff format
function parseDiff(diff: string): FileDiff[] {
  const fileDiffs: FileDiff[] = [];
  const diffLines = diff.split('\n');

  let currentFileDiff: FileDiff | null = null;
  let currentHunk: Hunk | null = null;
  let leftLineNumber = 0;
  let rightLineNumber = 0;

  for (const line of diffLines) {
    if (line.startsWith('diff --git')) {
      if (currentFileDiff) {
        if (currentHunk) {
          currentFileDiff.hunks.push(currentHunk);
        }
        fileDiffs.push(currentFileDiff);
      }
      const parts = line.split(' ');
      const fileName = parts.length > 2 ? parts[2].substring(2) : 'unknown';
      currentFileDiff = { fileName, hunks: [] };
      currentHunk = null;
      continue;
    }

    if (line.startsWith('---') || line.startsWith('+++')) {
        continue;
    }

    if (line.startsWith('@@')) {
      if (currentHunk && currentFileDiff) {
        currentFileDiff.hunks.push(currentHunk);
      }
      const match = /@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/.exec(line);
      if (match) {
        leftLineNumber = parseInt(match[1], 10);
        rightLineNumber = parseInt(match[2], 10);
      } else {
        leftLineNumber = 0;
        rightLineNumber = 0;
      }
      currentHunk = { header: line, linePairs: [] };
      continue;
    }

    if (!currentHunk) continue;

    if (line.startsWith('+')) {
      currentHunk.linePairs.push({
        right: { content: line.substring(1), type: 'add', lineNumber: rightLineNumber++ },
      });
    } else if (line.startsWith('-')) {
      currentHunk.linePairs.push({
        left: { content: line.substring(1), type: 'remove', lineNumber: leftLineNumber++ },
      });
    } else if (line.startsWith(' ')) {
      currentHunk.linePairs.push({
        left: { content: line.substring(1), type: 'context', lineNumber: leftLineNumber++ },
        right: { content: line.substring(1), type: 'context', lineNumber: rightLineNumber++ },
      });
    }
  }

  if (currentFileDiff) {
    if (currentHunk) {
      currentFileDiff.hunks.push(currentHunk);
    }
    fileDiffs.push(currentFileDiff);
  }

  return fileDiffs;
}


interface SideBySideDiffViewerProps {
  diff: string;
}

export function SideBySideDiffViewer({ diff }: SideBySideDiffViewerProps) {
  const theme = useTheme();
  const fileDiffs = parseDiff(diff);

  if (fileDiffs.length === 0) {
    return <Typography sx={{ p: 2 }}>No changes to display.</Typography>;
  }

  const lineStyle = {
    fontFamily: 'monospace',
    fontSize: '0.875rem',
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-all',
    minHeight: '1.2em',
    px: 1,
  };

  const lineNumberStyle = {
    color: theme.palette.text.secondary,
    textAlign: 'right' as const,
    pr: 1,
    userSelect: 'none' as const,
    width: '40px',
    flexShrink: 0,
  };

  return (
    <Box sx={{ p: 1, overflow: 'auto', height: '100%', bgcolor: 'background.default' }}>
      {fileDiffs.map((fileDiff, fileIndex) => (
        <Box key={fileIndex} sx={{ mb: 2, border: 1, borderColor: 'divider', borderRadius: 1, overflow: 'hidden' }}>
          <Typography sx={{ p: 1, bgcolor: 'action.hover', fontWeight: 'bold', fontFamily: 'monospace' }}>
            {fileDiff.fileName}
          </Typography>
          {fileDiff.hunks.map((hunk, hunkIndex) => (
            <Box key={hunkIndex}>
              <Typography sx={{ p: 1, bgcolor: 'action.hover', color: 'text.secondary', fontFamily: 'monospace', fontSize: '0.8rem' }}>
                {hunk.header}
              </Typography>
              {hunk.linePairs.map((pair, pairIndex) => (
                <Box key={pairIndex} sx={{ display: 'flex' }}>
                  {/* Left side */}
                  <Box sx={{
                    display: 'flex',
                    width: '50%',
                    backgroundColor: pair.left?.type === 'remove' ? alpha(theme.palette.error.main, 0.15) : 'transparent',
                    borderRight: 1,
                    borderColor: 'divider',
                  }}>
                    <Box sx={lineNumberStyle}>{pair.left?.lineNumber}</Box>
                    <Box sx={{ ...lineStyle, flexGrow: 1 }}>{pair.left?.content}</Box>
                  </Box>
                  {/* Right side */}
                  <Box sx={{
                    display: 'flex',
                    width: '50%',
                    backgroundColor: pair.right?.type === 'add' ? alpha(theme.palette.success.main, 0.15) : 'transparent',
                  }}>
                    <Box sx={lineNumberStyle}>{pair.right?.lineNumber}</Box>
                    <Box sx={{ ...lineStyle, flexGrow: 1 }}>{pair.right?.content}</Box>
                  </Box>
                </Box>
              ))}
            </Box>
          ))}
        </Box>
      ))}
    </Box>
  );
}
