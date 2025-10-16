import { Link, useParams } from 'react-router-dom'

const skills = [
  { name: 'ML Basics', val: 0.8 },
  { name: 'Computer Vision', val: 0.7 },
  { name: 'NLP', val: 0.6 },
  { name: 'MLOps', val: 0.5 },
  { name: 'Math/Stats', val: 0.75 },
]

export default function Results() {
  const { id } = useParams()

  return (
    <div className="min-h-screen bg-gradient-to-b from-orange-50 to-white">
      <header className="border-b border-orange-100 bg-white/70 backdrop-blur">
        <div className="max-w-6xl mx-auto px-4 py-4 flex items-center justify-between">
          <Link to="/dashboard" className="text-sm text-orange-600 hover:underline">← К дашборду</Link>
          <div className="font-semibold">Результаты: {id}</div>
        </div>
      </header>
      <main className="max-w-5xl mx-auto px-4 py-8 grid gap-6">
        <section className="bg-white rounded-xl border border-orange-100 p-6">
          <h2 className="text-xl font-semibold mb-2">Итоговая оценка</h2>
          <div className="grid md:grid-cols-2 gap-6">
            <div>
              <div className="text-3xl font-bold text-orange-700">74%</div>
              <p className="text-sm text-zinc-600">Стабильный Middle. Рекомендуем фокус на регуляризацию и валидацию.</p>
            </div>
            <div className="grid gap-3">
              {skills.map(s => (
                <div key={s.name}>
                  <div className="flex items-center justify-between text-sm">
                    <span>{s.name}</span>
                    <span className="text-zinc-500">{Math.round(s.val*100)}%</span>
                  </div>
                  <div className="h-2 bg-orange-100 rounded-full overflow-hidden">
                    <div className="h-full bg-gradient-to-r from-orange-400 to-rose-400" style={{ width: `${s.val*100}%` }} />
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="bg-white rounded-xl border border-orange-100 p-6">
          <h3 className="font-semibold mb-2">Рекомендации</h3>
          <ul className="list-disc pl-5 text-sm text-zinc-700 space-y-1">
            <li>Перепройти раздел Regularization: L1/L2, Dropout, Early stopping</li>
            <li>Практика валидации: Stratified k-fold, Leakage checks</li>
            <li>Усилить понимание метрик для несбалансированных данных (PR-AUC)</li>
          </ul>
        </section>

        <section className="bg-white rounded-xl border border-orange-100 p-6">
          <h3 className="font-semibold mb-2">График прогресса</h3>
          <div className="h-40 bg-gradient-to-t from-orange-100 to-white rounded-xl border border-orange-100 flex items-end gap-2 p-3">
            {[50, 62, 59, 71, 74].map((v, i) => (
              <div key={i} className="flex-1 bg-gradient-to-t from-orange-400 to-rose-300 rounded" style={{ height: `${v}%` }} />
            ))}
          </div>
        </section>
      </main>
    </div>
  )
}


