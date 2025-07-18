#!/bin/bash

# Функция для синхронизации якорей и заголовков
sync_anchors() {
    local file=$1
    local temp_file="${file}.tmp"
    
    # Обрабатываем файл построчно
    while IFS= read -r line; do
        # Исправляем HTML-якоря (удаляем лишние слова "type" и "func")
        if [[ "$line" == *"<a name="* ]]; then
            line=$(echo "$line" | sed -E 's/<a name="(type|func)?\.?([^"]+)"><\/a>/<a id="\2"><\/a>/')
        fi
        
        # Исправляем заголовки (удаляем "type" и "func")
        if [[ "$line" == "## "* ]]; then
            line=$(echo "$line" | sed -E 's/^## (type|func) ?//')
        fi
        
        echo "$line"
    done < "$file" > "$temp_file"
    
    mv "$temp_file" "$file"
    echo "Якоря и заголовки синхронизированы"
}

# Проверяем и запускаем
if [ -f "DOCUMENTATION.md" ]; then
    echo "Синхронизация документации..."
    sync_anchors "DOCUMENTATION.md"
else
    echo "Ошибка: файл DOCUMENTATION.md не найден"
    exit 1
fi