import React from "react"
import { createRoot } from "react-dom/client"
import Dashboard from "@medusajs/dashboard"
import "@medusajs/dashboard/css"

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <Dashboard />
  </React.StrictMode>
)
