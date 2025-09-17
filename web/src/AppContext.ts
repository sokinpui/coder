import { createContext } from 'react';

export interface AppContextType {
  toggleColorMode: () => void;
  codeTheme: 'light' | 'dark';
  toggleCodeTheme: () => void;
}

export const AppContext = createContext<AppContextType>({
  toggleColorMode: () => {},
  codeTheme: 'dark',
  toggleCodeTheme: () => {},
});
