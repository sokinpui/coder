export interface Message {
  sender: 'User' | 'AI' | 'System' | 'Command' | 'Result' | 'Error';
  content: string;
}
