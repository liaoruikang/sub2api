UPDATE accounts
SET extra = COALESCE(extra, '{}'::jsonb)
    - 'highest_scheduling_recovery_minutes'
    - 'highest_scheduling_suppressed'
    - 'highest_scheduling_suppressed_until'
    - 'highest_scheduling_suppressed_at'
    - 'highest_scheduling_suppressed_reason'
WHERE COALESCE(extra, '{}'::jsonb) ?| ARRAY[
    'highest_scheduling_recovery_minutes',
    'highest_scheduling_suppressed',
    'highest_scheduling_suppressed_until',
    'highest_scheduling_suppressed_at',
    'highest_scheduling_suppressed_reason'
];
