import { createContext } from 'react';

export interface AppContextType {
  toggleColorMode: () => void;
}

export const AppContext = createContext<AppContextType>({ toggleColorMode: () => {} });
