import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { getUserAccounts, getUserCategories, getUserTransactions, type Account, type Category, type Transaction } from '../api/admin'

export default function UserDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [txs, setTxs] = useState<Transaction[]>([])
  const [cats, setCats] = useState<Category[]>([])
  const [accs, setAccs] = useState<Account[]>([])
  const [error, setError] = useState('')
  const [tab, setTab] = useState<'transactions' | 'categories' | 'accounts'>('transactions')

  useEffect(() => {
    if (!id) return
    Promise.all([getUserTransactions(id), getUserCategories(id), getUserAccounts(id)])
      .then(([t, c, a]) => { setTxs(t); setCats(c); setAccs(a) })
      .catch(err => setError(err.response?.data?.error || '加载失败'))
  }, [id])

  if (error) return <div style={{ color: 'var(--danger)' }}>{error}</div>

  const catMap = new Map(cats.map(c => [c.id, c]))
  const accMap = new Map(accs.map(a => [a.id, a]))

  return (
    <div className="animate-fade-in-up">
      <div style={{ marginBottom: 24 }}>
        <Link to="/admin/users" style={{ display: 'inline-flex', alignItems: 'center', gap: 8, color: 'var(--text-secondary)', fontSize: 14 }}>
          <span>&larr;</span> 返回用户列表
        </Link>
      </div>
      
      <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 32 }}>
        <div style={{ width: 48, height: 48, borderRadius: 24, background: 'var(--primary-gradient)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 20, boxShadow: 'var(--shadow-glow)' }}>
          <span style={{ color: 'white' }}>👤</span>
        </div>
        <div>
          <h1 style={{ fontSize: 28, margin: 0, fontFamily: 'Outfit, sans-serif' }}>用户数据中心</h1>
          <div style={{ color: 'var(--text-secondary)', fontSize: 13, marginTop: 4 }}>ID: #{id}</div>
        </div>
      </div>

      <div style={{ display: 'flex', gap: 12, marginBottom: 24 }}>
        {([
          ['transactions', `交易记录 (${txs.length})`],
          ['categories', `全部分类 (${cats.length})`],
          ['accounts', `关联账户 (${accs.length})`]
        ] as const).map(([key, label]) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={tab === key ? 'primary' : ''}
            style={tab !== key ? { background: 'var(--surface)', border: '1px solid var(--border-light)' } : {}}
          >{label}</button>
        ))}
      </div>

      <div className="glass-panel animate-fade-in-up stagger-1" style={{ overflow: 'hidden' }}>
        {tab === 'transactions' && (
          <div className="table-container">
            <table>
              <thead>
                <tr><th>日期</th><th>类型</th><th>金额</th><th>分类</th><th>账户</th><th>备注</th></tr>
              </thead>
              <tbody>
                {txs.map(t => (
                  <tr key={t.id}>
                    <td style={{ color: 'var(--text-secondary)' }}>{t.date}{t.time ? ' ' + t.time : ''}</td>
                    <td>
                      <span className={t.type === 'income' ? 'badge badge-user' : t.type === 'expense' ? 'badge badge-admin' : 'badge'} style={t.type === 'transfer' ? { background: 'rgba(255,255,255,0.1)', color: 'white', border: '1px solid rgba(255,255,255,0.2)' } : {}}>
                        {t.type === 'income' ? '收入' : t.type === 'expense' ? '支出' : '转账'}
                      </span>
                    </td>
                    <td style={{ 
                      color: t.type === 'income' ? 'var(--success)' : t.type === 'expense' ? '#FCA5A5' : 'inherit',
                      fontFamily: 'Outfit, sans-serif',
                      fontWeight: 600
                    }}>
                      {t.type === 'income' ? '+' : t.type === 'expense' ? '-' : ''}{t.amount}
                    </td>
                    <td>{t.categoryId ? catMap.get(t.categoryId)?.name || '-' : '-'}</td>
                    <td>{t.accountId ? accMap.get(t.accountId)?.name || '-' : '-'}</td>
                    <td style={{ color: 'var(--text-secondary)' }}>{t.note || '-'}</td>
                  </tr>
                ))}
                {txs.length === 0 && <tr><td colSpan={6} style={{ textAlign: 'center', padding: 48, color: 'var(--text-tertiary)' }}>暂无交易记录</td></tr>}
              </tbody>
            </table>
          </div>
        )}
        {tab === 'categories' && (
          <div className="table-container">
            <table>
              <thead>
                <tr><th>名称</th><th>类型</th><th>图标</th><th>颜色</th><th>排序</th><th>系统</th></tr>
              </thead>
              <tbody>
                {cats.map(c => (
                  <tr key={c.id}>
                    <td style={{ fontWeight: 500 }}>{c.name}</td>
                    <td>{c.type === 'income' ? '收入' : '支出'}</td>
                    <td><span style={{ fontSize: 18 }}>{c.icon}</span></td>
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <span style={{ display: 'inline-block', width: 20, height: 20, background: c.colorHex, borderRadius: '50%', border: '2px solid rgba(255,255,255,0.1)' }} />
                        <span style={{ color: 'var(--text-secondary)', fontSize: 12, fontFamily: 'Outfit, sans-serif' }}>{c.colorHex}</span>
                      </div>
                    </td>
                    <td style={{ color: 'var(--text-secondary)' }}>{c.sortOrder}</td>
                    <td>
                      <span className={c.isSystem ? 'badge' : ''} style={c.isSystem ? { background: 'rgba(255,255,255,0.1)', color: 'white', border: '1px solid rgba(255,255,255,0.2)' } : { color: 'var(--text-tertiary)' }}>
                        {c.isSystem ? '系统内置' : '自定义'}
                      </span>
                    </td>
                  </tr>
                ))}
                {cats.length === 0 && <tr><td colSpan={6} style={{ textAlign: 'center', padding: 48, color: 'var(--text-tertiary)' }}>暂无分类</td></tr>}
              </tbody>
            </table>
          </div>
        )}
        {tab === 'accounts' && (
          <div className="table-container">
            <table>
              <thead>
                <tr><th>名称</th><th>类型</th><th>初始余额</th><th>排序</th><th>系统</th></tr>
              </thead>
              <tbody>
                {accs.map(a => (
                  <tr key={a.id}>
                    <td style={{ fontWeight: 500 }}><span style={{ marginRight: 8 }}>{a.icon}</span> {a.name}</td>
                    <td style={{ color: 'var(--text-secondary)' }}>{a.type}</td>
                    <td style={{ fontFamily: 'Outfit, sans-serif' }}>{a.initialBalance}</td>
                    <td style={{ color: 'var(--text-secondary)' }}>{a.sortOrder}</td>
                    <td>
                      <span className={a.isSystem ? 'badge' : ''} style={a.isSystem ? { background: 'rgba(255,255,255,0.1)', color: 'white', border: '1px solid rgba(255,255,255,0.2)' } : { color: 'var(--text-tertiary)' }}>
                        {a.isSystem ? '系统内置' : '自定义'}
                      </span>
                    </td>
                  </tr>
                ))}
                {accs.length === 0 && <tr><td colSpan={5} style={{ textAlign: 'center', padding: 48, color: 'var(--text-tertiary)' }}>暂无账户</td></tr>}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
