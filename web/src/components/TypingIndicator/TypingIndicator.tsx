import { Box, keyframes, styled } from '@mui/material';

const bounce = keyframes`
  0%, 80%, 100% {
    transform: scale(0);
  }
  40% {
    transform: scale(1.0);
  }
`;

const Dot = styled(Box)(({ theme }) => ({
  width: 8,
  height: 8,
  backgroundColor: theme.palette.text.secondary,
  borderRadius: '50%',
  display: 'inline-block',
  margin: '0 2px',
  animation: `${bounce} 1.4s infinite ease-in-out both`,
}));

export function TypingIndicator() {
  return (
    <Box sx={{ display: 'flex', alignItems: 'center' }}>
      <Dot sx={{ animationDelay: '-0.32s' }} />
      <Dot sx={{ animationDelay: '-0.16s' }} />
      <Dot />
    </Box>
  );
}
