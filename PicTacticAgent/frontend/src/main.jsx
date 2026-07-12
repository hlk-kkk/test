import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import './index.css'
import App from './App.jsx'
import { LocaleProvider } from '@/hooks/useLocale'
import { AuthProvider, ProtectedRoute, PublicOnlyRoute } from '@/hooks/useAuth'
import { useTheme } from '@/hooks/useTheme'
import ErrorBoundary from '@/components/ErrorBoundary'
import LoginPage from '@/pages/LoginPage'
import RegisterPage from '@/pages/RegisterPage'
import OAuthCallback from '@/pages/OAuthCallback'
import ForgotPasswordPage from '@/pages/ForgotPasswordPage'
import ResetPasswordPage from '@/pages/ResetPasswordPage'
import VerifyEmailPage from '@/pages/VerifyEmailPage'
import LandingPage from '@/pages/LandingPage'
import TemplatesPage from '@/pages/TemplatesPage'
import { Toaster } from '@/components/ui/Toaster'

function AppRoutes() {
  useTheme();

  return (
    <Routes>
      <Route path="/" element={<PublicOnlyRoute><LandingPage /></PublicOnlyRoute>} />
      <Route path="/auth/login" element={<PublicOnlyRoute><LoginPage /></PublicOnlyRoute>} />
      <Route path="/auth/register" element={<PublicOnlyRoute><RegisterPage /></PublicOnlyRoute>} />
      <Route path="/auth/callback/:provider" element={<OAuthCallback />} />
      <Route path="/auth/forgot-password" element={<PublicOnlyRoute><ForgotPasswordPage /></PublicOnlyRoute>} />
      <Route path="/auth/reset-password" element={<ResetPasswordPage />} />
      <Route path="/auth/verify-email" element={<VerifyEmailPage />} />
      <Route path="/app/templates" element={<ProtectedRoute><TemplatesPage /></ProtectedRoute>} />
      <Route path="/app/*" element={<ProtectedRoute><App /></ProtectedRoute>} />
    </Routes>
  );
}

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <ErrorBoundary>
      <LocaleProvider>
        <BrowserRouter>
          <AuthProvider>
            <AppRoutes />
            <Toaster />
          </AuthProvider>
        </BrowserRouter>
      </LocaleProvider>
    </ErrorBoundary>
  </StrictMode>,
)
