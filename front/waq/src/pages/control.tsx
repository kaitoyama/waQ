"use client"

import { useEffect, useState } from "react"
import { backendURL } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader } from "@/components/ui/card"

type Broadcast = { id: string; title: string; url: string; status: string; streamStatus: string }
type Controls = { actor: string; broadcasts: Broadcast[] }

export default function ControlPage() {
  const [controls, setControls] = useState<Controls | null>(null)
  const [message, setMessage] = useState("")
  const [running, setRunning] = useState<string | null>(null)

  const load = async () => {
    const response = await fetch(`${backendURL}/controls/broadcasts`)
    if (!response.ok) throw new Error("配信一覧を取得できませんでした")
    setControls(await response.json())
  }

  useEffect(() => { load().catch((error: Error) => setMessage(error.message)) }, [])

  const transition = async (broadcast: Broadcast, action: "start" | "stop") => {
    if (action === "stop" && !window.confirm(`「${broadcast.title}」を終了します。よろしいですか？`)) return
    setRunning(`${broadcast.id}:${action}`)
    setMessage("")
    try {
      const response = await fetch(`${backendURL}/controls/broadcasts/${encodeURIComponent(broadcast.id)}/${action}`, { method: "POST" })
      const body = await response.json()
      if (!response.ok) throw new Error(body.error || "操作に失敗しました")
      setMessage(action === "start" ? "配信を開始しました。" : "配信を終了しました。")
      await load()
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "操作に失敗しました")
    } finally {
      setRunning(null)
    }
  }

  return <main className="min-h-screen bg-slate-50 p-6"><Card className="mx-auto max-w-3xl"><CardHeader><h1 className="text-2xl font-bold">YouTube 配信操作</h1><p>操作ユーザー: {controls?.actor || "確認中"}</p></CardHeader><CardContent><p role="status" aria-live="polite" className="mb-4">{message}</p>{controls?.broadcasts.map((broadcast) => { const startDisabled = broadcast.status === "live" || broadcast.status === "testing" || broadcast.status === "liveStarting" || broadcast.streamStatus !== "active"; const stopDisabled = broadcast.status !== "live"; return <section key={broadcast.id} className="mb-4 rounded border p-4"><h2 className="font-semibold">{broadcast.title}</h2><a className="text-blue-700 underline" href={broadcast.url} target="_blank" rel="noreferrer">YouTube で開く</a><dl className="mt-2"><dt className="inline font-medium">配信状態: </dt><dd className="inline">{broadcast.status}</dd><br /><dt className="inline font-medium">ストリーム状態: </dt><dd className="inline">{broadcast.streamStatus || "未接続"}</dd></dl><div className="mt-3 flex gap-2"><Button disabled={startDisabled || running !== null} onClick={() => transition(broadcast, "start")}>{running === `${broadcast.id}:start` ? "開始中…" : "開始"}</Button><Button variant="destructive" disabled={stopDisabled || running !== null} onClick={() => transition(broadcast, "stop")}>{running === `${broadcast.id}:stop` ? "終了中…" : "終了"}</Button></div></section>})}</CardContent></Card></main>
}
