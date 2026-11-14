import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { AppRoutes } from './App'

describe('AppRoutes', () => {
  it('открывает консультанта на главном маршруте', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <AppRoutes />
      </MemoryRouter>,
    )

    expect(screen.getByRole('heading', { name: /опишите задачу/i })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Обзор' })).not.toBeInTheDocument()
  })

  it('показывает в футере кейс, команду и год', () => {
    render(
      <MemoryRouter initialEntries={['/assistant']}>
        <AppRoutes />
      </MemoryRouter>,
    )

    expect(screen.getByText('ИИ-консультант для бизнеса')).toBeInTheDocument()
    expect(screen.getByText('AIblanki')).toBeInTheDocument()
    expect(screen.getByText('2026')).toBeInTheDocument()
  })

  it.each(['/assistant', '/404'])('не упоминает главную страницу на маршруте %s', (route) => {
    render(
      <MemoryRouter initialEntries={[route]}>
        <AppRoutes />
      </MemoryRouter>,
    )

    expect(screen.queryByRole('link', { name: /главн|обзор/i })).not.toBeInTheDocument()
    expect(screen.queryByText(/главн|обзор/i)).not.toBeInTheDocument()
  })

  it('показывает 404 для неизвестного маршрута', () => {
    render(
      <MemoryRouter initialEntries={['/unknown-page']}>
        <AppRoutes />
      </MemoryRouter>,
    )

    expect(screen.getByText('ERROR / 404')).toBeInTheDocument()
  })

  it('проводит основной сценарий консультации и показывает источники', async () => {
    render(
      <MemoryRouter initialEntries={['/assistant']}>
        <AppRoutes />
      </MemoryRouter>,
    )

    const input = screen.getByLabelText('Сообщение консультанту')
    fireEvent.change(input, { target: { value: 'Хотим автоматический учёт показаний' } })
    fireEvent.submit(input.closest('form')!)

    await waitFor(() => expect(screen.getByText(/какие модели счётчиков/i)).toBeInTheDocument())
    fireEvent.click(screen.getByRole('button', { name: /открыть режим отладки/i }))
    expect(screen.getByText('22-askue.md')).toBeInTheDocument()
    expect(screen.getByText(/время ответа/i)).toBeInTheDocument()
  })

  it('открывает форму передачи заявки менеджеру', () => {
    render(
      <MemoryRouter initialEntries={['/assistant']}>
        <AppRoutes />
      </MemoryRouter>,
    )

    fireEvent.click(screen.getByRole('button', { name: /передать менеджеру/i }))
    expect(screen.getByRole('heading', { name: /оставить контакт/i })).toBeInTheDocument()
  })
})
