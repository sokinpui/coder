import { useMemo } from 'react';
import { Box, Typography, Chip, useTheme, Link } from '@mui/material';
import type { GitGraphLogEntry } from '../../types';

interface GitGraphProps {
  log: GitGraphLogEntry[];
  onCommitSelect: (hash: string) => void;
}

const ROW_HEIGHT = 28;
const COL_WIDTH = 18;
const DOT_RADIUS = 4;

const branchColors = [
  '#e6194B', '#3cb44b', '#ffe119', '#4363d8', '#f58231', '#911eb4', '#46f0f0',
  '#f032e6', '#bcf60c', '#fabebe', '#008080', '#e6beff', '#9A6324', '#fffac8',
  '#800000', '#aaffc3', '#808000', '#ffd8b1', '#000075', '#a9a9a9'
];

function calculateLayout(log: GitGraphLogEntry[]) {
  const commitMap = new Map<string, { commit: GitGraphLogEntry; rowIndex: number }>();
  log.forEach((commit, index) => {
    commitMap.set(commit.hash, { commit, rowIndex: index });
  });

  const lanes: (string | null)[] = [];
  const commitLanes = new Map<string, number>();
  const branchColorsMap = new Map<number, string>();
  let colorCounter = 0;

  log.forEach((commit) => {
    let laneIndex = lanes.indexOf(commit.hash);
    if (laneIndex === -1) {
      laneIndex = lanes.findIndex(l => l === null);
      if (laneIndex === -1) {
        laneIndex = lanes.length;
      }
    }
    commitLanes.set(commit.hash, laneIndex);

    // After using a lane for a commit, if other lanes were also pointing to this commit (merge),
    // they should be freed up.
    for (let i = 0; i < lanes.length; i++) {
      if (lanes[i] === commit.hash && i !== laneIndex) {
        lanes[i] = null;
      }
    }

    if (!branchColorsMap.has(laneIndex)) {
      branchColorsMap.set(laneIndex, branchColors[colorCounter % branchColors.length]);
      colorCounter++;
    }

    const parentHashes = commit.parentHashes;
    if (parentHashes.length > 0) {
      lanes[laneIndex] = parentHashes[0];
      parentHashes.slice(1).forEach(pHash => {
        let parentLaneIndex = lanes.indexOf(pHash);
        if (parentLaneIndex === -1) {
          parentLaneIndex = lanes.findIndex(l => l === null);
          if (parentLaneIndex === -1) {
            parentLaneIndex = lanes.length;
          }
          lanes[parentLaneIndex] = pHash;
        }
      });
    } else {
      lanes[laneIndex] = null;
    }
  });

  return { commitLanes, branchColorsMap, commitMap, maxLanes: lanes.length };
}

export function GitGraph({ log, onCommitSelect }: GitGraphProps) {
  const theme = useTheme();
  const { commitLanes, branchColorsMap, commitMap, maxLanes } = useMemo(() => calculateLayout(log), [log]);

  const graphWidth = maxLanes * COL_WIDTH;
  const graphHeight = log.length * ROW_HEIGHT;

  return (
    <Box sx={{ position: 'relative', fontFamily: 'monospace', fontSize: '0.875rem', overflow: 'auto', height: '100%' }}>
      <svg width={graphWidth} height={graphHeight} style={{ position: 'absolute', top: 0, left: 0, zIndex: 0 }}>
        {/* Render all paths */}
        {log.flatMap((commit, rowIndex) => {
          const commitLane = commitLanes.get(commit.hash);
          if (commitLane === undefined) return [];
          const cx = commitLane * COL_WIDTH + COL_WIDTH / 2;
          const cy = rowIndex * ROW_HEIGHT + ROW_HEIGHT / 2;

          return (commit.parentHashes || []).map(pHash => {
            const parent = commitMap.get(pHash);
            if (!parent) return null;
            const parentLane = commitLanes.get(pHash);
            if (parentLane === undefined) return null;

            const pcx = parentLane * COL_WIDTH + COL_WIDTH / 2;
            const pcy = parent.rowIndex * ROW_HEIGHT + ROW_HEIGHT / 2;
            const parentColor = branchColorsMap.get(parentLane) || theme.palette.text.primary;

            const isMerge = parentLane !== commitLane;
            const d = isMerge
              ? `M ${cx} ${cy} C ${cx} ${(cy + pcy) / 2}, ${pcx} ${(cy + pcy) / 2}, ${pcx} ${pcy}`
              : `M ${cx} ${cy} L ${pcx} ${pcy}`;

            return <path key={`${commit.hash}-${pHash}`} d={d} stroke={parentColor} fill="none" strokeWidth="2" />;
          }).filter(Boolean);
        })}
        {/* Render all circles on top of paths */}
        {log.map((commit, rowIndex) => {
          const commitLane = commitLanes.get(commit.hash);
          if (commitLane === undefined) return null;
          const cx = commitLane * COL_WIDTH + COL_WIDTH / 2;
          const cy = rowIndex * ROW_HEIGHT + ROW_HEIGHT / 2;
          const color = branchColorsMap.get(commitLane) || theme.palette.text.primary;
          return <circle key={commit.hash} cx={cx} cy={cy} r={DOT_RADIUS} fill={color} />;
        })}
      </svg>
      {/* Render commit info on top of the SVG */}
      <Box sx={{ position: 'relative', zIndex: 1 }}>
        {log.map((commit) => (
          <Box key={commit.hash} sx={{ display: 'flex', height: ROW_HEIGHT, alignItems: 'center' }}>
            <Box sx={{ width: graphWidth, flexShrink: 0 }} /> {/* Spacer */}
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'nowrap', overflow: 'hidden', pl: 1 }}>
              <Link
                component="button"
                variant="body2"
                onClick={() => onCommitSelect(commit.hash)}
                sx={{
                  fontFamily: 'monospace',
                  textAlign: 'left',
                  textDecoration: 'none',
                  '&:hover': { textDecoration: 'underline' },
                  whiteSpace: 'nowrap',
                }}
              >
                {commit.subject}
              </Link>
              <Chip label={commit.hash.substring(0, 7)} size="small" variant="outlined" />
              {commit.refs?.map(ref => (
                <Chip key={ref} label={ref} size="small" color="secondary" sx={{
                  fontWeight: ref.includes('HEAD') ? 'bold' : 'normal'
                }} />
              ))}
              <Typography variant="caption" color="text.secondary" noWrap sx={{ ml: 'auto', pl: 1 }}>
                {commit.authorName}, {commit.relativeDate}
              </Typography>
            </Box>
          </Box>
        ))}
      </Box>
    </Box>
  );
}
