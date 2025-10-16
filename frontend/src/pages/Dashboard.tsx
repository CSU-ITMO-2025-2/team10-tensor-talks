import { Link, useNavigate } from 'react-router-dom'
import { useState } from 'react'
import MVPNotification from '../components/MVPNotification'

function Card({ children }: { children: React.ReactNode }) {
  return <div className="bg-white rounded-xl border border-orange-100 shadow-soft p-5">{children}</div>
}

export default function Dashboard() {
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<'learner' | 'hr'>('learner')
  const [showMVPPopup, setShowMVPPopup] = useState(false)
  
  const handleFeatureClick = () => {
    setShowMVPPopup(true)
  }
  
  const interviews = [
    { id: 'ml-101', title: 'ML System Design', date: '12.09.2025', score: 78 },
    { id: 'ml-102', title: 'NLP Basics', date: '05.09.2025', score: 64 },
  ]
  const candidates = [
    { id: 'cand-1', name: 'Алексей Петров', role: 'ML Engineer', score: 72 },
    { id: 'cand-2', name: 'Мария Иванова', role: 'DS/ML', score: 81 },
  ]
  const recommendations = [
    { topic: 'Регуляризация и обобщающая способность', level: 'Middle', eta: '25 мин' },
    { topic: 'Оптимизация гиперпараметров (Grid/Random/Bayes)', level: 'Middle', eta: '35 мин' },
    { topic: 'Проектирование фичей для табличных данных', level: 'Junior→Middle', eta: '40 мин' },
  ]
  const recentActivity = [
    { id: 'act-1', label: 'Завершено интервью: ML System Design', when: '2 дня назад' },
    { id: 'act-2', label: 'Добавлена заметка к вопросу: Bias-Variance', when: '4 дня назад' },
    { id: 'act-3', label: 'Обновлён план: Метрические функции', when: 'неделю назад' },
  ]

  return (
    <div className="min-h-screen bg-gradient-to-b from-orange-50 to-white">
      <MVPNotification isOpen={showMVPPopup} onClose={() => setShowMVPPopup(false)} />
      <header className="border-b border-orange-100 bg-white/70 backdrop-blur">
        <div className="max-w-6xl mx-auto px-4 py-4 flex items-center">
          <Link to="/" className="flex items-center gap-3">
            <div className="size-8 rounded-lg bg-gradient-to-br from-orange-500 to-rose-500" />
            <div className="font-bold">TensorTalks</div>
          </Link>
          <div className="flex-1 flex justify-center">
            <div className="rounded-xl border border-orange-100 bg-white p-1 inline-flex gap-1 text-sm">
              <button 
                onClick={() => setActiveTab('learner')} 
                className={`px-3 py-1.5 rounded-lg ${activeTab === 'learner' ? 'bg-orange-600 text-white' : 'hover:bg-orange-50'}`}
              >
                Для обучаемого
              </button>
              <button 
                onClick={() => setActiveTab('hr')} 
                className={`px-3 py-1.5 rounded-lg ${activeTab === 'hr' ? 'bg-orange-600 text-white' : 'hover:bg-orange-50'}`}
              >
                Для HR/TechLead
              </button>
            </div>
          </div>
          <nav className="flex items-center gap-4 text-sm">
            <button onClick={handleFeatureClick} className="text-orange-700 hover:underline cursor-pointer">Профиль</button>
            <button onClick={handleFeatureClick} className="text-zinc-600 hover:underline cursor-pointer">Подписка</button>
            <button onClick={() => navigate('/auth')} className="px-3 py-1.5 rounded-lg border border-orange-200 hover:bg-orange-50">Выйти</button>
          </nav>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-4 py-8">
        <div className="mb-6">
          <h1 className="text-2xl md:text-3xl font-bold tracking-tight">Дашборд</h1>
          <p className="text-zinc-600 mt-1">
            {activeTab === 'learner' 
              ? 'Ваш личный центр подготовки к ML‑интервью: прогресс, рекомендации и быстрый старт.'
              : 'Обзор воронки кандидатов, активных интервью и аналитики компетенций команды.'}
          </p>
        </div>
        {/* Табы перенесены в хедер */}

        {/* Верхние карточки - условно отображаются в зависимости от активного таба */}
        {activeTab === 'learner' && (
          <div className="grid md:grid-cols-3 gap-5 mb-6">
            <Card>
              <div className="text-sm text-zinc-500">Профиль уровня</div>
              <div className="text-lg font-semibold">Middle ML · индивидуальный план</div>
              <div className="text-sm text-zinc-500">Сильные стороны: CV, Метрики • Зона роста: Регуляризация</div>
            </Card>
            <Card>
              <div className="text-sm text-zinc-500">Прогресс</div>
              <div className="mt-2 h-2 bg-orange-100 rounded-full overflow-hidden">
                <div className="h-full w-2/3 bg-gradient-to-r from-orange-400 to-rose-400" />
              </div>
              <div className="text-sm mt-2">67% выполнения недельного плана</div>
            </Card>
            <Card>
              <div className="text-sm text-zinc-500">Подписка</div>
              <div className="font-semibold">Pro · 990 ₽/мес</div>
              <button onClick={handleFeatureClick} className="mt-2 px-3 py-1.5 rounded-lg bg-orange-600 text-white hover:bg-orange-700">Управлять подпиской</button>
            </Card>
          </div>
        )}

        {activeTab === 'hr' && (
          <div className="grid md:grid-cols-3 gap-5 mb-6">
            <Card>
              <div className="text-sm text-zinc-500">Команда</div>
              <div className="text-lg font-semibold">HR · TechLead</div>
              <div className="text-sm text-zinc-500">Управление кандидатами и интервью • Доступ к отчётам</div>
            </Card>
            <Card>
              <div className="text-sm text-zinc-500">Воронка</div>
              <div className="mt-2 h-2 bg-orange-100 rounded-full overflow-hidden">
                <div className="h-full w-4/5 bg-gradient-to-r from-orange-400 to-rose-400" />
              </div>
              <div className="text-sm mt-2">12 активных интервью в этом месяце</div>
            </Card>
            <Card>
              <div className="text-sm text-zinc-500">Подписка</div>
              <div className="font-semibold">Enterprise · 2990 ₽/мес</div>
              <button onClick={handleFeatureClick} className="mt-2 px-3 py-1.5 rounded-lg bg-orange-600 text-white hover:bg-orange-700">Управление</button>
            </Card>
          </div>
        )}

        {/* Контент для обучаемого */}
        {activeTab === 'learner' && (
          <section className="grid gap-6">
            {/* Быстрые действия */}
            <div className="flex flex-wrap gap-3">
              <button onClick={handleFeatureClick} className="px-4 py-2 rounded-lg bg-gradient-to-r from-orange-600 to-rose-600 text-white hover:from-orange-700 hover:to-rose-700">
                Начать новое интервью
              </button>
              <button onClick={handleFeatureClick} className="px-4 py-2 rounded-lg border border-orange-200 hover:bg-orange-50">
                Посмотреть отчёты
              </button>
              <button onClick={handleFeatureClick} className="px-4 py-2 rounded-lg border border-orange-200 hover:bg-orange-50">
                План подготовки
              </button>
            </div>

            {/* Рекомендовано к изучению */}
            <div>
              <h2 className="text-xl font-semibold mb-3">Рекомендовано к изучению</h2>
              <div className="grid md:grid-cols-3 gap-4">
                {recommendations.map((r) => (
                  <Card key={r.topic}>
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="font-medium">{r.topic}</div>
                        <div className="text-sm text-zinc-500">{r.eta} • {r.level}</div>
                      </div>
                      <span className="text-lg">📚</span>
                    </div>
                    <button onClick={handleFeatureClick} className="mt-3 px-3 py-1.5 rounded-lg bg-orange-600 text-white hover:bg-orange-700 text-sm">Пройти тренировку</button>
                  </Card>
                ))}
              </div>
            </div>

            {/* Пройденные интервью */}
            <div>
              <h2 className="text-xl font-semibold mb-3">Пройденные интервью</h2>
              <div className="overflow-hidden rounded-xl border border-orange-100">
                <table className="w-full text-sm">
                  <thead className="bg-orange-50">
                    <tr>
                      <th className="text-left p-3">Интервью</th>
                      <th className="text-left p-3">Дата</th>
                      <th className="text-left p-3">Оценка</th>
                    </tr>
                  </thead>
                  <tbody>
                    {interviews.map((i) => (
                      <tr key={i.id} className="border-t border-orange-100 hover:bg-orange-50/50">
                        <td className="p-3 text-orange-700 underline cursor-pointer" onClick={() => navigate(`/chat/${i.id}`)}>{i.title}</td>
                        <td className="p-3">{i.date}</td>
                        <td className="p-3 text-orange-700 underline cursor-pointer" onClick={() => navigate(`/results/${i.id}`)}>{i.score}%</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            {/* Недавняя активность */}
            <div>
              <h2 className="text-xl font-semibold mb-3">Недавняя активность</h2>
              <div className="grid md:grid-cols-3 gap-4">
                {recentActivity.map((a) => (
                  <Card key={a.id}>
                    <div className="flex items-start justify-between">
                      <div className="text-sm">
                        <div className="font-medium text-zinc-900">{a.label}</div>
                        <div className="text-zinc-500 mt-0.5">{a.when}</div>
                      </div>
                      <span>🕒</span>
                    </div>
                  </Card>
                ))}
              </div>
            </div>
          </section>
        )}

        {/* Контент для HR/TechLead */}
        {activeTab === 'hr' && (
          <section className="grid gap-6">
            {/* Быстрые действия для HR */}
            <div className="flex flex-wrap gap-3">
              <button onClick={handleFeatureClick} className="px-4 py-2 rounded-lg bg-gradient-to-r from-orange-600 to-rose-600 text-white hover:from-orange-700 hover:to-rose-700">
                Создать интервью кандидату
              </button>
              <button onClick={handleFeatureClick} className="px-4 py-2 rounded-lg border border-orange-200 hover:bg-orange-50">
                Открыть отчёты по команде
              </button>
              <button onClick={handleFeatureClick} className="px-4 py-2 rounded-lg border border-orange-200 hover:bg-orange-50">
                Управление доступами
              </button>
            </div>

            {/* Кандидаты */}
            <div>
              <h2 className="text-xl font-semibold mb-3">Кандидаты</h2>
              <div className="overflow-hidden rounded-xl border border-orange-100">
                <table className="w-full text-sm">
                  <thead className="bg-orange-50">
                    <tr>
                      <th className="text-left p-3">Кандидат</th>
                      <th className="text-left p-3">Роль</th>
                      <th className="text-left p-3">Сводка</th>
                    </tr>
                  </thead>
                  <tbody>
                    {candidates.map((c) => (
                      <tr key={c.id} className="border-t border-orange-100 hover:bg-orange-50/50">
                        <td className="p-3 text-orange-700 underline cursor-pointer" onClick={() => navigate(`/chat/${c.id}`)}>{c.name}</td>
                        <td className="p-3">{c.role}</td>
                        <td className="p-3 text-orange-700 underline cursor-pointer" onClick={() => navigate(`/results/${c.id}`)}>{c.score}% пригодности</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            {/* Блоки аналитики */}
            <div className="grid md:grid-cols-3 gap-4">
              <Card>
                <div className="text-sm text-zinc-500">Калибровка интервьюеров</div>
                <div className="text-lg font-semibold">Отклонение оценок · 6%</div>
                <div className="text-sm text-zinc-500">Цель ≤ 5% • рекомендуем синхронизацию критериев</div>
              </Card>
              <Card>
                <div className="text-sm text-zinc-500">Среднее время интервью</div>
                <div className="text-lg font-semibold">38 мин</div>
                <div className="text-sm text-zinc-500">AI‑интервью экономит ~22 мин по сравнению с ручным</div>
              </Card>
              <Card>
                <div className="text-sm text-zinc-500">Конверсия этапа</div>
                <div className="text-lg font-semibold">Техническое → оффер: 18%</div>
                <div className="text-sm text-zinc-500">Следите за узкими местами по компетенциям</div>
              </Card>
            </div>
          </section>
        )}
      </main>
    </div>
  )
}


