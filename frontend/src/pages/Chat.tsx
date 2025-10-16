import { Link, useParams } from 'react-router-dom'
import { useState } from 'react'
import MVPNotification from '../components/MVPNotification'

export default function Chat() {
  const { id } = useParams()
  const [showMVPPopup, setShowMVPPopup] = useState(false)
  
  const handleFeatureClick = () => {
    setShowMVPPopup(true)
  }
  const items = [
    { q: 'Объясните разницу между L1 и L2 регуляризацией.', a: 'L1 ведет к разреженности весов, L2 — к их уменьшению. L1 добавляет |w|, L2 — w^2 к функции потерь.' },
    { q: 'Как работает кросс‑валидация k-fold?', a: 'Данные делятся на k фолдов, обучаемся на k-1 и валидируем на оставшемся; повторяем и усредняем.' },
    { q: 'Фрагмент кода: вычисление ROC‑AUC', a: 'См. пример ниже.' },
  ]

  return (
    <div className="min-h-screen bg-gradient-to-b from-orange-50 to-white">
      <MVPNotification isOpen={showMVPPopup} onClose={() => setShowMVPPopup(false)} />
      <header className="border-b border-orange-100 bg-white/70 backdrop-blur">
        <div className="max-w-6xl mx-auto px-4 py-4 flex items-center justify-between">
          <Link to="/dashboard" className="text-sm text-orange-600 hover:underline">← К дашборду</Link>
          <div className="font-semibold">Сессия: {id}</div>
        </div>
      </header>
      <main className="max-w-4xl mx-auto px-4 py-8 grid gap-4">
        <div className="bg-white rounded-xl border border-orange-100 p-4">
          <div className="text-sm text-zinc-500 mb-2">Чат с AI‑интервьюером</div>
          <div className="space-y-4">
            {items.map((m, i) => (
              <div key={i} className="grid gap-2">
                <div className="self-start max-w-[80%] rounded-2xl px-4 py-2 bg-orange-100 text-zinc-900">Вопрос: {m.q}</div>
                <div className="self-start max-w-[80%] rounded-2xl px-4 py-2 bg-white border border-orange-100 shadow-soft">Ответ: {m.a}</div>
              </div>
            ))}
            <div className="mt-2">
              <div className="text-sm text-zinc-500 mb-1">Код:</div>
              <pre className="bg-zinc-950 text-zinc-100 rounded-lg p-4 overflow-auto text-xs">
{`import numpy as np
from sklearn.metrics import roc_auc_score

y_true = np.array([0, 1, 1, 0, 1])
y_score = np.array([0.2, 0.9, 0.6, 0.4, 0.8])
print('ROC-AUC:', roc_auc_score(y_true, y_score))`}
              </pre>
            </div>
          </div>
        </div>
        <div className="flex gap-2">
          <input 
            className="flex-1 px-3 py-2 rounded-lg border border-orange-200" 
            placeholder="Напишите ответ..." 
            onFocus={handleFeatureClick}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                handleFeatureClick()
              }
            }}
          />
          <button onClick={handleFeatureClick} className="px-4 py-2 rounded-lg bg-orange-600 text-white hover:bg-orange-700">Отправить</button>
        </div>
      </main>
    </div>
  )
}


