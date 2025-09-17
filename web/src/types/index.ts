export interface Message {
  sender: "User" | "AI" | "System" | "Command" | "Result" | "Error";
  content: string;
}

export interface HistoryItem {
  filename: string;
  title: string;
  modifiedAt: string; // ISO string
}

export interface SourceNode {
  name: string;
  path: string;
  type: "file" | "directory";
  children?: SourceNode[];
}
