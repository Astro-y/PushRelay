import type { ReactNode } from "react"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import { InboxIcon } from "lucide-react"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { useI18n } from "@/lib/i18n"

export type Column<T> = {
  key: string
  label: string
  render: (item: T) => ReactNode
}
export function ResourceTable<T extends { id: string }>({
  items,
  columns,
}: {
  items: T[]
  columns: Column<T>[]
}) {
  const { t } = useI18n()
  if (!items.length)
    return (
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <InboxIcon />
          </EmptyMedia>
          <EmptyTitle>{t("暂无数据")}</EmptyTitle>
          <EmptyDescription>
            {t("使用页面右上角的按钮创建第一条记录。")}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  return (
    <Table>
      <TableHeader>
        <TableRow>
          {columns.map((column) => (
            <TableHead key={column.key}>{t(column.label)}</TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => (
          <TableRow key={item.id}>
            {columns.map((column) => (
              <TableCell key={column.key}>{column.render(item)}</TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
