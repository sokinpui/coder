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

export interface GitLogEntry {
  hash: string;
  authorName: string;
  relativeDate: string;
  subject: string;
  body: string;
}
