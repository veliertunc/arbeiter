export type User = {
  id?: string;
  email?: string;
  name?: string;
  email_verified?: boolean;
};

export type Organization = {
  id?: string;
  name?: string;
  slug?: string;
  role?: string;
};

export type ConsoleTab = "Overview" | "Profile" | "Organizations" | "Sessions";
