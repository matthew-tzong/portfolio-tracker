// Environment variables for the frontend.
interface ImportMetaEnv {
  readonly VITE_SUPABASE_URL: string
  readonly VITE_SUPABASE_ANON_KEY: string
  readonly VITE_API_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

// Types for Plaid Link Integration.
interface PlaidLinkOnSuccessMetadata {
  institution?: {
    name?: string
    institution_id?: string
  }
  accounts?: Array<{
    id: string
    name?: string
    mask?: string
    type?: string
    subtype?: string
  }>
}

interface PlaidLinkHandler {
  open: () => void
  destroy: () => void
}

interface PlaidLinkOptions {
  token: string
  onSuccess: (publicToken: string, metadata: PlaidLinkOnSuccessMetadata) => void
  onExit?: (err: unknown | null, metadata: unknown) => void
}

interface PlaidGlobal {
  create: (options: PlaidLinkOptions) => PlaidLinkHandler
}

interface Window {
  Plaid?: PlaidGlobal
}
