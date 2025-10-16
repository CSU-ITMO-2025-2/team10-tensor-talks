import { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'

export default function Auth() {
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const navigate = useNavigate()
  const [isLoaded, setIsLoaded] = useState(false)

  useEffect(() => {
    // Анимация появления
    const timer = setTimeout(() => setIsLoaded(true), 100)
    return () => clearTimeout(timer)
  }, [])

  // Красивые примеры для автозаполнения
  const sampleData = {
    names: ['Александр Волков', 'Мария Петрова', 'Дмитрий Козлов', 'Анна Соколова', 'Иван Морозов'],
    emails: ['alex.volkov@tech.com', 'maria.petrova@ml.dev', 'dmitry.kozlov@ai.startup', 'anna.sokolova@data.science', 'ivan.morozov@research.org'],
    passwords: ['SecurePass123!', 'ML_Expert_2025', 'DataScience#99', 'AI_Researcher_88', 'TechLead_2025!']
  }

  const getRandomSample = (array: string[]) => array[Math.floor(Math.random() * array.length)]

  return (
    <div className="min-h-screen bg-gradient-to-br from-orange-50 via-white to-rose-50 relative overflow-hidden">
      {/* Декоративные элементы */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute -top-40 -right-40 w-80 h-80 bg-gradient-to-tr from-orange-300/20 to-rose-300/20 rounded-full blur-3xl animate-pulse" />
        <div className="absolute -bottom-40 -left-40 w-80 h-80 bg-gradient-to-tr from-blue-300/20 to-purple-300/20 rounded-full blur-3xl" />
        <div className="absolute top-1/2 left-1/4 w-40 h-40 bg-gradient-to-tr from-green-300/20 to-blue-300/20 rounded-full blur-2xl animate-bounce" />
        <div className="absolute top-1/4 right-1/3 w-32 h-32 bg-gradient-to-tr from-purple-300/20 to-pink-300/20 rounded-full blur-2xl animate-pulse" />
      </div>

      <div className="relative z-10 flex items-center justify-center min-h-screen p-6">
        <div className={`w-full max-w-xl bg-white/80 backdrop-blur-sm rounded-3xl border border-orange-100 shadow-2xl p-6 transition-all duration-1000 transform ${isLoaded ? 'translate-y-0 opacity-100' : 'translate-y-8 opacity-0'}`}>
          <div className="flex items-center justify-between mb-3">
            <Link to="/" className="text-sm text-orange-600 hover:text-orange-700 hover:underline transition-colors duration-200 flex items-center gap-1">
              <span>←</span> На главную
            </Link>
            <div className="flex items-center gap-2">
              <div className="w-6 h-6 rounded-lg bg-gradient-to-br from-orange-500 to-rose-500" />
              <span className="text-sm font-semibold text-zinc-700">TensorTalks</span>
            </div>
          </div>

          <div className="text-center mb-3">
            <div className="mx-auto size-12 rounded-2xl bg-gradient-to-tr from-orange-500 to-rose-500 shadow-lg flex items-center justify-center mb-2" style={{animation: 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite'}}>
              <span className="text-white text-2xl">🧠</span>
            </div>
            <h1 className="text-xl font-bold text-zinc-900 mb-1">
              {mode === 'login' ? 'Добро пожаловать!' : 'Присоединяйтесь к нам'}
            </h1>
            <p className="text-sm text-zinc-500 mb-2">
              AI‑симулятор технических ML‑собеседований
            </p>
            
            {/* Краткая информация о продукте */}
            <div className="bg-gradient-to-r from-orange-50 to-rose-50 rounded-xl p-1.5 border border-orange-100">
              <div className="flex items-center justify-center gap-6 text-xs text-zinc-600">
                <div className="flex items-center gap-1">
                  <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
                  Объективная оценка
                </div>
                <div className="flex items-center gap-1">
                  <span className="w-2 h-2 bg-blue-500 rounded-full animate-pulse"></span>
                  AI-интервьюер
                </div>
                <div className="flex items-center gap-1">
                  <span className="w-2 h-2 bg-purple-500 rounded-full animate-pulse"></span>
                  Персональные рекомендации
                </div>
              </div>
            </div>
          </div>

          <form className="grid gap-2.5" onSubmit={(e) => { e.preventDefault(); navigate('/dashboard') }}>
            {mode === 'register' && (
              <label className="grid gap-2">
                <span className="text-sm font-medium text-zinc-700 pl-4">Полное имя</span>
                <input 
                  className="px-4 py-3 rounded-xl border border-zinc-200 bg-white hover:border-orange-300 focus:border-orange-500 focus:ring-2 focus:ring-orange-100 transition-all duration-200" 
                  placeholder={getRandomSample(sampleData.names)}
                  defaultValue={getRandomSample(sampleData.names)}
                />
              </label>
            )}
            <label className="grid gap-2">
              <span className="text-sm font-medium text-zinc-700 pl-4">Email</span>
              <input 
                type="email" 
                className="px-4 py-3 rounded-xl border border-zinc-200 bg-white hover:border-orange-300 focus:border-orange-500 focus:ring-2 focus:ring-orange-100 transition-all duration-200" 
                placeholder={getRandomSample(sampleData.emails)}
                defaultValue={getRandomSample(sampleData.emails)}
              />
            </label>
            <label className="grid gap-2">
              <span className="text-sm font-medium text-zinc-700 pl-4">Пароль</span>
              <input 
                type="password" 
                className="px-4 py-3 rounded-xl border border-zinc-200 bg-white hover:border-orange-300 focus:border-orange-500 focus:ring-2 focus:ring-orange-100 transition-all duration-200" 
                placeholder="••••••••••"
                defaultValue={getRandomSample(sampleData.passwords)}
              />
            </label>
            
            
            <button className="px-6 py-3 rounded-xl bg-gradient-to-r from-orange-600 to-rose-600 text-white font-semibold hover:from-orange-700 hover:to-rose-700 shadow-lg hover:shadow-xl transform hover:scale-102 transition-all duration-300">
              {mode === 'login' ? 'Войти в систему' : 'Создать аккаунт'}
            </button>
          </form>

          <div className="mt-3 text-center">
            <div className="flex items-center gap-2 justify-center text-xs text-zinc-500">
              <div className="h-px bg-zinc-200 flex-1" />
              <span className="shrink-0">или</span>
              <div className="h-px bg-zinc-200 flex-1" />
            </div>
            
            <div className="mt-4 text-sm text-zinc-600">
              {mode === 'login' ? (
                <button 
                  className="text-orange-600 hover:text-orange-700 hover:underline font-medium transition-colors duration-200" 
                  onClick={() => setMode('register')}
                >
                  Нет аккаунта? Зарегистрироваться
                </button>
              ) : (
                <button 
                  className="text-orange-600 hover:text-orange-700 hover:underline font-medium transition-colors duration-200" 
                  onClick={() => setMode('login')}
                >
                  Уже есть аккаунт? Войти
                </button>
              )}
            </div>
          </div>

          {/* Дополнительная информация */}
          <div className="mt-3 pt-2 border-t border-zinc-100">
            <div className="text-center">
              <p className="text-xs text-zinc-500 mb-3">Что вас ждет после регистрации:</p>
              <div className="flex justify-center">
                <div className="grid grid-cols-2 gap-x-6 gap-y-2 text-xs text-zinc-600 max-w-md">
                  <div className="flex items-start gap-2">
                    <span className="text-green-500 flex-shrink-0 mt-0.5">✓</span>
                    <span className="leading-tight">Доступ к базе вопросов</span>
                  </div>
                  <div className="flex items-start gap-2">
                    <span className="text-green-500 flex-shrink-0 mt-0.5">✓</span>
                    <span className="leading-tight">AI-разбор ваших ответов</span>
                  </div>
                  <div className="flex items-start gap-2">
                    <span className="text-green-500 flex-shrink-0 mt-0.5">✓</span>
                    <span className="leading-tight">Персональные рекомендации</span>
                  </div>
                  <div className="flex items-start gap-2">
                    <span className="text-green-500 flex-shrink-0 mt-0.5">✓</span>
                    <span className="leading-tight">Отслеживание прогресса</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}


