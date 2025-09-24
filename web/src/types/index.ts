export interface Message {
  sender: "User" | "AI" | "System" | "Command" | "Result" | "Error" | "Image";
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

export interface GitGraphLogEntry {
  hash: string;
  parentHashes: string[];
  authorName: string;
  relativeDate: string;
  subject: string;
  refs: string[];
}
