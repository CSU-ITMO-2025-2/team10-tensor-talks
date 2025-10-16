import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createHashRouter, RouterProvider, Navigate } from 'react-router-dom'
import './index.css'
import App from './App.tsx'
import Auth from './pages/Auth.tsx'
import Dashboard from './pages/Dashboard.tsx'
import Chat from './pages/Chat.tsx'
import Results from './pages/Results.tsx'

const router = createHashRouter([
  { path: '/', element: <App /> },
  { path: '/auth', element: <Auth /> },
  { path: '/dashboard', element: <Dashboard /> },
  { path: '/chat/:id', element: <Chat /> },
  { path: '/results/:id', element: <Results /> },
  { path: '*', element: <Navigate to="/" replace /> },
])

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>,
)
