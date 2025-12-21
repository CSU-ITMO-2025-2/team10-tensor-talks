import { Link, useParams, useNavigate } from 'react-router-dom'
import { useState, useEffect } from 'react'
import { getInterviewResult, getInterviewChat, type ChatMessage } from '../services/chat'

export default function Results() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [result, setResult] = useState<any>(null)
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const loadData = async () => {
      if (!id) {
        console.error('Results: no session id in URL')
        return
      }

      console.log('Results: loading data for session:', id)
      try {
        setIsLoading(true)
        
        console.log('Results: calling getInterviewResult and getInterviewChat')
        const [resultData, chatHistory] = await Promise.all([
          getInterviewResult(id).catch((err) => {
            console.error('Results: getInterviewResult error:', err)
            return null
          }),
          getInterviewChat(id).catch((err) => {
            console.error('Results: getInterviewChat error:', err)
            // Если чат не найден, используем пустой массив, но логируем ошибку
            return []
          })
        ])
        
        console.log('Results: received data', { resultData, chatHistoryLength: chatHistory?.length })
        
        if (resultData) {
          setResult(resultData)
        }
        if (chatHistory && chatHistory.length > 0) {
          setMessages(chatHistory)
        } else {
          console.warn('Results: Chat history is empty or not found')
          setMessages([])
        }
      } catch (error) {
        console.error('Results: Failed to load results:', error)
        alert('Не удалось загрузить результаты. Попробуйте еще раз.')
        navigate('/dashboard')
      } finally {
        setIsLoading(false)
      }
    }

    loadData()
  }, [id, navigate])

  return (
    <div className="min-h-screen bg-gradient-to-b from-orange-50 to-white">
      <header className="border-b border-orange-100 bg-white/70 backdrop-blur">
        <div className="max-w-6xl mx-auto px-4 py-4 flex items-center justify-between">
          <Link to="/dashboard" className="text-sm text-orange-600 hover:underline">← К дашборду</Link>
          <div className="font-semibold">Результаты: {id}</div>
        </div>
      </header>
      <main className="max-w-5xl mx-auto px-4 py-8 grid gap-6">
        {isLoading ? (
          <section className="bg-white rounded-xl border border-orange-100 p-6">
            <div className="text-center text-zinc-500">Загрузка результатов...</div>
          </section>
        ) : result ? (
          <>
            <section className="bg-white rounded-xl border border-orange-100 p-6">
              <h2 className="text-xl font-semibold mb-2">Итоговая оценка</h2>
              <div className="grid md:grid-cols-2 gap-6">
                <div>
                  <div className="text-3xl font-bold text-orange-700">{result.score}%</div>
                  <p className="text-sm text-zinc-600 mt-2">{result.feedback}</p>
                </div>
              </div>
            </section>

            <section className="bg-white rounded-xl border border-orange-100 p-6">
              <h3 className="font-semibold mb-4">История чата</h3>
              <div className="space-y-4 max-h-96 overflow-y-auto">
                {messages.length === 0 ? (
                  <div className="text-sm text-zinc-500 text-center py-4">История чата недоступна</div>
                ) : (
                  messages.map((msg, idx) => (
                    <div 
                      key={idx} 
                      className={`p-3 rounded-lg ${
                        msg.type === 'system' 
                          ? 'bg-orange-50 border border-orange-100' 
                          : 'bg-zinc-50 border border-zinc-100'
                      }`}
                    >
                      <div className="text-xs text-zinc-500 mb-1">
                        {msg.type === 'system' ? 'Система' : 'Вы'} • {new Date(msg.created_at).toLocaleString('ru-RU')}
                      </div>
                      <div className="text-sm text-zinc-900">{msg.content}</div>
                    </div>
                  ))
                )}
              </div>
            </section>
          </>
        ) : (
          <section className="bg-white rounded-xl border border-orange-100 p-6">
            <div className="text-center text-zinc-500">Результаты не найдены</div>
          </section>
        )}
      </main>
    </div>
  )
}


