import type React from "react"

import { useState } from "react"
import { Calendar, Upload, X, Copy, Check } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { Textarea } from "@/components/ui/textarea"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { cn } from "@/lib/utils"
import { format } from "date-fns"
import { ja } from "date-fns/locale"
import { Calendar as CalendarComponent } from "@/components/ui/calendar"
import { zodResolver } from "@hookform/resolvers/zod"
import { useForm } from "react-hook-form"
import * as z from "zod"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { useToast } from "@/hooks/use-toast"
import Image from "next/image"
import { Switch } from "@/components/ui/switch"
import { FormDescription } from "@/components/ui/form"

const MAX_FILE_SIZE = 5 * 1024 * 1024 // 5MB
const ACCEPTED_IMAGE_TYPES = ["image/jpeg", "image/png", "image/webp"]

// formSchema の更新
const formSchema = z.object({
  title: z.string().min(1, "配信タイトルを入力してください"),
  startDate: z.date({
    required_error: "配信開始日時を選択してください",
  }),
  startTime: z
    .string({
      required_error: "配信開始時刻を選択してください",
    })
    .regex(/^([01]?[0-9]|2[0-3]):[0-5][0-9]$/, "正しい時刻を入力してください"),
  visibility: z.enum(["public", "private"], {
    required_error: "公開設定を選択してください",
  }),
  latency: z.enum(["ultra_low", "low", "normal"], {
    required_error: "遅延設定を選択してください",
  }),
  description: z.string(),
  thumbnail: z
    .any()
    .refine((files) => !files || files.length === 0 || files.length === 1, "サムネイル画像は1つだけ選択できます")
    .refine(
      (files) => !files || files.length === 0 || files[0].size <= MAX_FILE_SIZE,
      "ファイルサイズは5MB以下にしてください",
    )
    .refine(
      (files) => !files || files.length === 0 || ACCEPTED_IMAGE_TYPES.includes(files[0].type),
      "JPG、PNG、WebP形式のみ対応しています",
    )
    .optional(),
  autoStart: z.boolean().default(true),
  autoEnd: z.boolean().default(true),
})

export default function StreamForm() {
  const { toast } = useToast()
  const [streamInfo, setStreamInfo] = useState<{ key: string; url: string } | null>(null)
  const [thumbnailPreview, setThumbnailPreview] = useState<string | null>(null)
  const [thumbnailInputKey, setThumbnailInputKey] = useState<number>(0)
  const [copiedKey, setCopiedKey] = useState(false)
  const [copiedUrl, setCopiedUrl] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)

  // StreamForm コンポーネント内の form の defaultValues を更新
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      title: "",
      visibility: "public",
      latency: "normal",
      description: "",
      startTime: "00:00",
      autoStart: true,
      autoEnd: true,
    },
  })

  // onSubmit 関数内でフォームデータに新しいフィールドを追加
  async function onSubmit(values: z.infer<typeof formSchema>) {
    try {
      setIsSubmitting(true)
      const formData = new FormData()
      const [hours, minutes] = values.startTime.split(":").map(Number)
      const startDateTime = new Date(values.startDate)
      startDateTime.setHours(hours, minutes)

      // 過去の日時チェック
      if (startDateTime < new Date()) {
        toast({
          variant: "destructive",
          title: "エラー",
          description: "配信開始日時は現在時刻より後に設定してください",
        })
        return
      }

      Object.entries(values).forEach(([key, value]) => {
        if (key === "thumbnail" && value?.[0]) {
          formData.append(key, value[0])
        } else if (key === "startDate") {
          formData.append(key, startDateTime.toISOString())
        } else if (key !== "startTime") {
          formData.append(key, value as string)
        }
      })

      formData.append("autoStart", values.autoStart.toString())
      formData.append("autoEnd", values.autoEnd.toString())

      await new Promise((resolve) => setTimeout(resolve, 1000))
      setStreamInfo({
        key: "live_xxxxxxxxxxx",
        url: "https://youtube.com/live/xxxxxxxxxxx",
      })

      // スクロールをトップに移動
      window.scrollTo({ top: 0, behavior: "smooth" })

      toast({
        title: "設定を保存しました",
        description: "配信の準備が整いました",
      })
    } catch (error) {
      toast({
        variant: "destructive",
        title: "エラーが発生しました",
        description: "もう一度お試しください",
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleThumbnailChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      const reader = new FileReader()
      reader.onloadend = () => {
        setThumbnailPreview(reader.result as string)
      }
      reader.readAsDataURL(file)
    } else {
      setThumbnailPreview(null)
    }
  }

  const handleCopy = async (text: string, type: "key" | "url") => {
    await navigator.clipboard.writeText(text)
    if (type === "key") {
      setCopiedKey(true)
      setTimeout(() => setCopiedKey(false), 2000)
    } else {
      setCopiedUrl(true)
      setTimeout(() => setCopiedUrl(false), 2000)
    }
    toast({
      title: "コピーしました",
      description: `${type === "key" ? "ストリームキー" : "配信URL"}をクリップボードにコピーしました`,
    })
  }

  return (
    <Card className="w-full max-w-2xl mx-auto">
      <CardHeader className="pb-4">
        <div className="flex flex-col space-y-2">
          <div className="text-3xl font-bold bg-gradient-to-r from-blue-600 to-indigo-600 bg-clip-text text-transparent">
            waQ
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {streamInfo && (
          <Alert className="mb-6">
            <AlertTitle>配信情報</AlertTitle>
            <AlertDescription>
              <div className="mt-2 space-y-2">
                <div className="flex items-center gap-2">
                  <span className="font-medium">ストリームキー:</span>
                  <code className="relative rounded bg-muted px-[0.3rem] py-[0.2rem] font-mono text-sm flex-1 select-text cursor-text overflow-hidden text-ellipsis whitespace-nowrap">
                    {streamInfo.key}
                  </code>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 hover:bg-muted"
                    onClick={() => handleCopy(streamInfo.key, "key")}
                    title="ストリームキーをコピー"
                  >
                    {copiedKey ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
                  </Button>
                </div>
                <div className="flex items-center gap-2">
                  <span className="font-medium">配信URL:</span>
                  <code className="relative rounded bg-muted px-[0.3rem] py-[0.2rem] font-mono text-sm flex-1 select-text cursor-text overflow-hidden text-ellipsis whitespace-nowrap">
                    {streamInfo.url}
                  </code>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 hover:bg-muted"
                    onClick={() => handleCopy(streamInfo.url, "url")}
                    title="配信URLをコピー"
                  >
                    {copiedUrl ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
                  </Button>
                </div>
              </div>
            </AlertDescription>
          </Alert>
        )}

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
            <FormField
              control={form.control}
              name="thumbnail"
              render={({ field: { value, onChange, ...field } }) => (
                <FormItem>
                  <FormLabel>サムネイル (任意)</FormLabel>
                  <FormControl>
                    <div className="space-y-4">
                      <div className="flex items-center justify-center w-full">
                        {thumbnailPreview ? (
                          <div className="relative w-full h-64">
                            <Image
                              src={thumbnailPreview || "/placeholder.svg"}
                              alt="サムネイルプレビュー"
                              fill
                              className="object-contain"
                            />
                            <Button
                              type="button"
                              variant="destructive"
                              size="icon"
                              className="absolute top-2 right-2"
                              onClick={(e) => {
                                e.preventDefault()
                                e.stopPropagation()
                                setThumbnailPreview(null)
                                onChange(null)
                                setThumbnailInputKey((prev) => prev + 1)
                              }}
                            >
                              <X className="h-4 w-4" />
                            </Button>
                          </div>
                        ) : (
                          <label
                            htmlFor="thumbnail"
                            className="flex flex-col items-center justify-center w-full h-64 border-2 border-dashed rounded-lg cursor-pointer bg-gray-50 dark:hover:bg-bray-800 dark:bg-gray-700 hover:bg-gray-100 dark:border-gray-600 dark:hover:border-gray-500 dark:hover:bg-gray-600"
                          >
                            <div className="flex flex-col items-center justify-center pt-5 pb-6">
                              <Upload className="w-8 h-8 mb-4 text-gray-500 dark:text-gray-400" />
                              <p className="mb-2 text-sm text-gray-500 dark:text-gray-400">
                                クリックまたはドラッグ＆ドロップで画像をアップロード
                              </p>
                              <p className="text-xs text-gray-500 dark:text-gray-400">JPG, PNG, WebP (最大5MB)</p>
                            </div>
                          </label>
                        )}
                        <input
                          key={thumbnailInputKey}
                          id="thumbnail"
                          type="file"
                          className="hidden"
                          accept="image/jpeg,image/png,image/webp"
                          onChange={(e) => {
                            handleThumbnailChange(e)
                            onChange(e.target.files)
                          }}
                          {...field}
                        />
                      </div>
                    </div>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="title"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>配信タイトル</FormLabel>
                  <FormControl>
                    <Input placeholder="配信タイトルを入力" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="startDate"
              render={({ field }) => (
                <FormItem className="flex flex-col">
                  <FormLabel>配信開始日時</FormLabel>
                  <div className="space-y-2">
                    <div className="flex gap-4">
                      <Popover>
                        <PopoverTrigger asChild>
                          <FormControl>
                            <Button
                              variant={"outline"}
                              className={cn(
                                "w-[240px] pl-3 text-left font-normal",
                                !field.value && "text-muted-foreground",
                                "focus-visible:ring-2 focus-visible:ring-ring",
                              )}
                            >
                              {field.value ? (
                                format(field.value, "yyyy年MM月dd日 (E)", { locale: ja })
                              ) : (
                                <span>日付を選択</span>
                              )}
                              <Calendar className="ml-auto h-4 w-4 opacity-50" />
                            </Button>
                          </FormControl>
                        </PopoverTrigger>
                        <PopoverContent className="w-auto p-0" align="start">
                          <CalendarComponent
                            mode="single"
                            selected={field.value}
                            onSelect={(date) => {
                              field.onChange(date)
                              // 日付が変更された時に、その日の現在時刻以降の時間を設定
                              if (date) {
                                const now = new Date()
                                if (date.toDateString() === now.toDateString()) {
                                  const currentTime = `${String(now.getHours()).padStart(2, "0")}:${String(now.getMinutes()).padStart(2, "0")}`
                                  form.setValue("startTime", currentTime)
                                } else {
                                  form.setValue("startTime", "00:00")
                                }
                              }
                            }}
                            disabled={(date) => {
                              const today = new Date()
                              today.setHours(0, 0, 0, 0)
                              return date < today
                            }}
                            initialFocus
                            locale={ja}
                          />
                        </PopoverContent>
                      </Popover>

                      <FormField
                        control={form.control}
                        name="startTime"
                        render={({ field: timeField }) => (
                          <FormItem>
                            <FormControl>
                              <Input
                                type="time"
                                className={cn("w-[120px]", "focus-visible:ring-2 focus-visible:ring-ring")}
                                onChange={(e) => {
                                  const selectedDate = form.getValues("startDate")
                                  const [hours, minutes] = e.target.value.split(":").map(Number)
                                  const selectedDateTime = new Date(selectedDate)
                                  selectedDateTime.setHours(hours, minutes)

                                  if (selectedDate && selectedDateTime < new Date()) {
                                    const now = new Date()
                                    const currentTime = `${String(now.getHours()).padStart(2, "0")}:${String(now.getMinutes()).padStart(2, "0")}`
                                    timeField.onChange(currentTime)
                                  } else {
                                    timeField.onChange(e.target.value)
                                  }
                                }}
                                value={timeField.value}
                              />
                            </FormControl>
                          </FormItem>
                        )}
                      />
                    </div>
                    <FormMessage />
                  </div>
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="visibility"
              render={({ field }) => (
                <FormItem className="space-y-3">
                  <FormLabel>公開設定</FormLabel>
                  <FormControl>
                    <RadioGroup
                      onValueChange={field.onChange}
                      defaultValue={field.value}
                      className="flex flex-col space-y-1"
                    >
                      <FormItem className="flex items-center space-x-3 space-y-0">
                        <FormControl>
                          <RadioGroupItem value="public" />
                        </FormControl>
                        <FormLabel className="font-normal">公開</FormLabel>
                      </FormItem>
                      <FormItem className="flex items-center space-x-3 space-y-0">
                        <FormControl>
                          <RadioGroupItem value="private" />
                        </FormControl>
                        <FormLabel className="font-normal">限定公開</FormLabel>
                      </FormItem>
                    </RadioGroup>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="latency"
              render={({ field }) => (
                <FormItem className="space-y-3">
                  <FormLabel>遅延設定</FormLabel>
                  <FormControl>
                    <RadioGroup
                      onValueChange={field.onChange}
                      defaultValue={field.value}
                      className="flex flex-col space-y-1"
                    >
                      <FormItem className="flex items-center space-x-3 space-y-0">
                        <FormControl>
                          <RadioGroupItem value="ultra_low" />
                        </FormControl>
                        <FormLabel className="font-normal">超低遅延</FormLabel>
                      </FormItem>
                      <FormItem className="flex items-center space-x-3 space-y-0">
                        <FormControl>
                          <RadioGroupItem value="low" />
                        </FormControl>
                        <FormLabel className="font-normal">低遅延</FormLabel>
                      </FormItem>
                      <FormItem className="flex items-center space-x-3 space-y-0">
                        <FormControl>
                          <RadioGroupItem value="normal" />
                        </FormControl>
                        <FormLabel className="font-normal">通常</FormLabel>
                      </FormItem>
                    </RadioGroup>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>説明</FormLabel>
                  <FormControl>
                    <Textarea placeholder="配信の説明を入力" className="resize-none" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <Alert variant="warning" className="mt-4">
              <AlertTitle>自動開始・終了設定</AlertTitle>
              <AlertDescription>
                <div className="space-y-4">
                  <FormField
                    control={form.control}
                    name="autoStart"
                    render={({ field }) => (
                      <FormItem className="flex flex-row items-center justify-between space-y-0">
                        <div className="space-y-0.5">
                          <FormLabel className="text-base">自動開始</FormLabel>
                          <FormDescription>設定した開始時刻に自動的に配信を開始します</FormDescription>
                        </div>
                        <FormControl>
                          <Switch checked={field.value} onCheckedChange={field.onChange} />
                        </FormControl>
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="autoEnd"
                    render={({ field }) => (
                      <FormItem className="flex flex-row items-center justify-between space-y-0">
                        <div className="space-y-0.5">
                          <FormLabel className="text-base">自動終了</FormLabel>
                          <FormDescription>設定した終了時刻に自動的に配信を終了します</FormDescription>
                        </div>
                        <FormControl>
                          <Switch checked={field.value} onCheckedChange={field.onChange} />
                        </FormControl>
                      </FormItem>
                    )}
                  />
                  <p className="text-sm text-muted-foreground">
                    自動開始または自動終了をオフにする場合は、庶務による操作が必要です。設定を変更する前に必ず庶務にご相談ください。
                  </p>
                </div>
              </AlertDescription>
            </Alert>

            <Button type="submit" className="w-full" disabled={isSubmitting}>
              {isSubmitting ? (
                <div className="flex items-center gap-2">
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
                  処理中...
                </div>
              ) : (
                "作成"
              )}
            </Button>
          </form>
        </Form>
      </CardContent>
    </Card>
  )
}

