import { create } from "zustand";

interface AuthState {
  token: string | null;
  account: string | null;
  setSession: (token: string, account: string) => void;
  clear: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  account: null,
  setSession: (token, account) => set({ token, account }),
  clear: () => set({ token: null, account: null }),
}));
