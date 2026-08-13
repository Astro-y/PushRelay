import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { useI18n } from "@/lib/i18n"

const commonTimezones = [
  "UTC",
  "Asia/Shanghai",
  "Asia/Hong_Kong",
  "Asia/Tokyo",
  "Asia/Singapore",
  "Europe/London",
  "Europe/Berlin",
  "America/New_York",
  "America/Los_Angeles",
]

const intlWithTimezones = Intl as typeof Intl & {
  supportedValuesOf?: (key: "timeZone") => string[]
}

const supportedTimezones = Array.from(
  new Set([
    ...commonTimezones,
    ...(intlWithTimezones.supportedValuesOf?.("timeZone") ?? []),
  ])
).sort((left, right) => left.localeCompare(right, "en"))

type TimezoneComboboxProps = {
  id?: string
  value: string
  onValueChange: (value: string) => void
  invalid?: boolean
}

export function TimezoneCombobox({
  id,
  value,
  onValueChange,
  invalid = false,
}: TimezoneComboboxProps) {
  const { t } = useI18n()
  const items = supportedTimezones.includes(value)
    ? supportedTimezones
    : [value, ...supportedTimezones].filter(Boolean)

  return (
    <Combobox
      items={items}
      value={value}
      onValueChange={(nextValue) => {
        if (nextValue) onValueChange(nextValue)
      }}
      autoHighlight
    >
      <ComboboxInput
        id={id}
        aria-invalid={invalid}
        placeholder={t("搜索或选择 IANA 时区")}
      />
      <ComboboxContent>
        <ComboboxEmpty>{t("未找到匹配的时区")}</ComboboxEmpty>
        <ComboboxList>
          {(timezone) => (
            <ComboboxItem key={timezone} value={timezone}>
              {timezone}
            </ComboboxItem>
          )}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  )
}
