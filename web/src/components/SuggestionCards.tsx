import React from 'react'
import { Suggestion as SuggestionType } from '../types'

interface SuggestionCardsProps {
    suggestions: SuggestionType[]
    onSelect: (suggestion: SuggestionType) => void
    onNone: () => void
}

export const SuggestionCards: React.FC<SuggestionCardsProps> = ({ suggestions, onSelect, onNone }) => {
    const icons: Record<string, string> = {
        reuse: '📦',
        create: '➕',
        none: '⚙️',
    }

    if (suggestions.length === 0) {
        return (
            <div className="text-center text-gray-500 py-4">
                <p className="text-lg font-medium">No suggestions available</p>
                <p className="text-sm">Please specify what you want to create</p>
            </div>
        )
    }

    return (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
            {suggestions.map((suggestion) => (
                <button
                    key={suggestion.id || `${suggestion.type}-${suggestion.name}`}
                    onClick={() => onSelect(suggestion)}
                    className="p-4 bg-white rounded-xl border-2 border-gray-200 shadow-lg hover:border-primary-500 hover:shadow-xl transition-all text-left"
                >
                    <span className="text-3xl mr-3">{icons[suggestion.type] || '📦'}</span>

                    <div className="space-y-1">
                        <p className="font-semibold text-gray-900">
                            {suggestion.type === 'reuse' ? `Reuse: ${suggestion.name}` : `Create: ${suggestion.name}`}
                        </p>

                        {suggestion.existing && (
                            <div className="flex items-center gap-2 text-xs text-green-600">
                                <span>✓ Recently used</span>
                            </div>
                        )}

                        <p className="text-sm text-gray-600">
                            {suggestion.namespace}/{suggestion.resource}
                        </p>

                        <div className="pt-2 border-t border-gray-100">
                            <p className="text-xs text-gray-500 italic">
                                Why: {suggestion.reason}
                            </p>
                        </div>
                    </div>
                </button>
            ))}

            <button
                onClick={onNone}
                className="p-4 bg-gray-100 rounded-xl border-2 border-gray-300 hover:bg-gray-200 transition-all"
            >
                <span className="text-3xl mr-3">⚙️</span>
                <div>
                    <p className="font-semibold text-gray-700">Customize...</p>
                    <p className="text-sm text-gray-500">Specify your own configuration</p>
                </div>
            </button>
        </div>
    )
}
