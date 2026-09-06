import { setupServer } from "msw/node"
import { mockHandlers } from "./handlers"

export const mockServer = setupServer(...mockHandlers)
