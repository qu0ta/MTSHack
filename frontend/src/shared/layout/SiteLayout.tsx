import { Link, Outlet } from 'react-router-dom'
import styles from './SiteLayout.module.css'

export function SiteLayout() {
  return (
    <div className={styles.shell}>
      <header className={styles.header}>
        <Link className={styles.brand} to="/assistant" aria-label="МТС EnergyTool">
          <span className={styles.mark} aria-hidden="true"><span /></span>
          <span className={styles.brandText}>ENERGY<br />TOOL</span>
        </Link>
      </header>

      <main>
        <Outlet />
      </main>

      <footer className={styles.footer}>
        <span>2026</span>
        <span>ИИ-консультант для бизнеса</span>
        <span>Команда: <b>AIblanki</b></span>
      </footer>
    </div>
  )
}
