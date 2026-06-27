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
    <div>
      <div style={{ marginBottom: 16 }}>
        <Link to="/admin/users">← 返回用户列表</Link>
      </div>
      <h1 style={{ fontSize: 22, marginBottom: 16 }}>用户 #{id} 数据</h1>

      <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
        {([
          ['transactions', `交易 (${txs.length})`],
          ['categories', `分类 (${cats.length})`],
          ['accounts', `账户 (${accs.length})`]
        ] as const).map(([key, label]) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={tab === key ? 'primary' : ''}
          >{label}</button>
        ))}
      </div>

      <div style={{ background: 'var(--surface)', borderRadius: 8, overflow: 'hidden' }}>
        {tab === 'transactions' && (
          <table>
            <thead>
              <tr><th>日期</th><th>类型</th><th>金额</th><th>分类</th><th>账户</th><th>备注</th></tr>
            </thead>
            <tbody>
              {txs.map(t => (
                <tr key={t.id}>
                  <td>{t.date}{t.time ? ' ' + t.time : ''}</td>
                  <td>{t.type === 'income' ? '收入' : t.type === 'expense' ? '支出' : '转账'}</td>
                  <td style={{ color: t.type === 'income' ? 'var(--success)' : t.type === 'expense' ? 'var(--danger)' : 'inherit' }}>
                    {t.type === 'income' ? '+' : t.type === 'expense' ? '-' : ''}{t.amount}
                  </td>
                  <td>{t.categoryId ? catMap.get(t.categoryId)?.name || '-' : '-'}</td>
                  <td>{t.accountId ? accMap.get(t.accountId)?.name || '-' : '-'}</td>
                  <td>{t.note}</td>
                </tr>
              ))}
              {txs.length === 0 && <tr><td colSpan={6} style={{ textAlign: 'center', padding: 24, color: 'var(--text-dim)' }}>无交易</td></tr>}
            </tbody>
          </table>
        )}
        {tab === 'categories' && (
          <table>
            <thead>
              <tr><th>名称</th><th>类型</th><th>图标</th><th>颜色</th><th>排序</th><th>系统</th></tr>
            </thead>
            <tbody>
              {cats.map(c => (
                <tr key={c.id}>
                  <td>{c.name}</td>
                  <td>{c.type === 'income' ? '收入' : '支出'}</td>
                  <td>{c.icon}</td>
                  <td><span style={{ display: 'inline-block', width: 16, height: 16, background: c.colorHex, borderRadius: 3, verticalAlign: 'middle' }} /></td>
                  <td>{c.sortOrder}</td>
                  <td>{c.isSystem ? '是' : '否'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        {tab === 'accounts' && (
          <table>
            <thead>
              <tr><th>名称</th><th>类型</th><th>初始余额</th><th>排序</th><th>系统</th></tr>
            </thead>
            <tbody>
              {accs.map(a => (
                <tr key={a.id}>
                  <td>{a.icon} {a.name}</td>
                  <td>{a.type}</td>
                  <td>{a.initialBalance}</td>
                  <td>{a.sortOrder}</td>
                  <td>{a.isSystem ? '是' : '否'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
