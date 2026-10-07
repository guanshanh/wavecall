/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_DISPATCH_URL: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
