import { Navigate, Route, Routes } from 'react-router-dom'
import { SiteLayout } from '../shared/layout/SiteLayout'
import { AssistantPage } from '../pages/AssistantPage/AssistantPage'
import { NotFoundPage } from '../pages/NotFoundPage/NotFoundPage'

export function AppRoutes() {
  return (
    <Routes>
      <Route element={<SiteLayout />}>
        <Route index element={<Navigate to="/assistant" replace />} />
        <Route path="assistant" element={<AssistantPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  )
}
