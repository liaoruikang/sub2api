import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AffiliateWithdrawalAccountsDialog from '../AffiliateWithdrawalAccountsDialog.vue'

const {
  createAffiliateWithdrawalAccount,
  setDefaultAffiliateWithdrawalAccount,
} = vi.hoisted(() => ({
  createAffiliateWithdrawalAccount: vi.fn(),
  setDefaultAffiliateWithdrawalAccount: vi.fn(),
}))

vi.mock('@/api/user', () => ({
  default: {
    createAffiliateWithdrawalAccount,
    updateAffiliateWithdrawalAccount: vi.fn(),
    setDefaultAffiliateWithdrawalAccount,
    deleteAffiliateWithdrawalAccount: vi.fn(),
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

const baseDialogStub = {
  props: ['show'],
  template: '<div v-if="show"><slot /><slot name="footer" /></div>',
}

describe('AffiliateWithdrawalAccountsDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    createAffiliateWithdrawalAccount.mockResolvedValue({ id: 1 })
    setDefaultAffiliateWithdrawalAccount.mockResolvedValue({ id: 2 })
  })

  it('adds a saved account and refreshes the parent list', async () => {
    const wrapper = mount(AffiliateWithdrawalAccountsDialog, {
      props: { show: true, accounts: [] },
      global: {
        stubs: {
          BaseDialog: baseDialogStub,
          ConfirmDialog: true,
          Icon: true,
        },
      },
    })

    const addButton = wrapper.findAll('button').find((button) => button.text() === 'affiliate.withdrawal.accounts.add')
    expect(addButton).toBeTruthy()
    await addButton!.trigger('click')
    await wrapper.get('#affiliate-account-editor').setValue('buyer@example.com')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(createAffiliateWithdrawalAccount).toHaveBeenCalledWith('buyer@example.com')
    expect(wrapper.emitted('changed')).toHaveLength(1)
  })

  it('sets a non-default account as default', async () => {
    const wrapper = mount(AffiliateWithdrawalAccountsDialog, {
      props: {
        show: true,
        accounts: [
          {
            id: 1,
            user_id: 42,
            account_type: 'alipay',
            account_masked: '138****0000',
            is_default: true,
            created_at: '2026-08-16T00:00:00Z',
            updated_at: '2026-08-16T00:00:00Z',
          },
          {
            id: 2,
            user_id: 42,
            account_type: 'alipay',
            account_masked: 'buy****om',
            is_default: false,
            created_at: '2026-08-16T00:00:00Z',
            updated_at: '2026-08-16T00:00:00Z',
          },
        ],
      },
      global: {
        stubs: {
          BaseDialog: baseDialogStub,
          ConfirmDialog: true,
          Icon: true,
        },
      },
    })

    const defaultButton = wrapper.findAll('button').find((button) => button.text() === 'affiliate.withdrawal.accounts.setDefault')
    expect(defaultButton).toBeTruthy()
    await defaultButton!.trigger('click')
    await flushPromises()

    expect(setDefaultAffiliateWithdrawalAccount).toHaveBeenCalledWith(2)
    expect(wrapper.emitted('changed')).toHaveLength(1)
  })
})
