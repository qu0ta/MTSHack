import { Link } from 'react-router-dom'
import styles from './NotFoundPage.module.css'

export function NotFoundPage() {
  return (
    <section className={styles.page}>
      <span>ERROR / 404</span>
      <h1>Не нашли<br />такой маршрут<span>.</span></h1>
      <p>Такого адреса нет. Откройте консультанта и продолжите работу.</p>
      <div><Link to="/assistant">К консультанту</Link></div>
    </section>
  )
}
