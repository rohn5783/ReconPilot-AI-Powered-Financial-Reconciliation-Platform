import React, { useState, useMemo } from 'react';
import {
  Sliders,
  CheckCircle2,
  AlertCircle,
  Plus,
  Edit2,
  Trash2,
  Zap,
  Calculator,
  ShieldCheck,
  Scale,
  RefreshCw,
  Search,
  Percent,
  Layers,
  ArrowRight,
  Info,
  Check,
  X,
  FileCheck,
} from 'lucide-react';
import { useReconStore } from '../../store/useReconStore';
import { ContractRule } from '../../types';

export const ContractRulesView: React.FC = () => {
  const {
    contractRules,
    isLoadingRules,
    fetchContractRules,
    saveContractRule,
    activateContractRule,
    deleteContractRule,
  } = useReconStore();

  const [search, setSearch] = useState('');
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<ContractRule | null>(null);

  // Form State
  const [formName, setFormName] = useState('');
  const [formMDRRate, setFormMDRRate] = useState('2.0000');
  const [formGSTRate, setFormGSTRate] = useState('18.0000');
  const [formFeeTolerance, setFormFeeTolerance] = useState('0.0100');
  const [formPrecision, setFormPrecision] = useState<number>(4);
  const [formRoundingMode, setFormRoundingMode] = useState<string>('HALF_EVEN');
  const [formDescription, setFormDescription] = useState('');
  const [formIsActive, setFormIsActive] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Simulation test amount
  const [simAmount, setSimAmount] = useState('10000.00');

  // Currently active rule
  const activeRule = useMemo(() => {
    return (contractRules || []).find((r) => r.is_active) || null;
  }, [contractRules]);

  // Filtered rules
  const filteredRules = useMemo(() => {
    return (contractRules || []).filter((r) => {
      const q = search.toLowerCase();
      return (
        r.name.toLowerCase().includes(q) ||
        r.description.toLowerCase().includes(q) ||
        r.rounding_mode.toLowerCase().includes(q)
      );
    });
  }, [contractRules, search]);

  // Open modal for Create
  const handleOpenCreate = () => {
    setEditingRule(null);
    setFormName('');
    setFormMDRRate('2.0000');
    setFormGSTRate('18.0000');
    setFormFeeTolerance('0.0100');
    setFormPrecision(4);
    setFormRoundingMode('HALF_EVEN');
    setFormDescription('');
    setFormIsActive(false);
    setFormError(null);
    setIsModalOpen(true);
  };

  // Open modal for Edit
  const handleOpenEdit = (rule: ContractRule) => {
    setEditingRule(rule);
    setFormName(rule.name);
    // Parse rates into percentage format if stored as decimal
    const mdrVal = parseFloat(rule.mdr_rate);
    setFormMDRRate(mdrVal > 0 && mdrVal < 1 ? (mdrVal * 100).toFixed(4) : rule.mdr_rate);
    const gstVal = parseFloat(rule.gst_rate);
    setFormGSTRate(gstVal > 0 && gstVal < 1 ? (gstVal * 100).toFixed(4) : rule.gst_rate);
    setFormFeeTolerance(rule.fee_tolerance);
    setFormPrecision(rule.rounding_precision || 4);
    setFormRoundingMode(rule.rounding_mode || 'HALF_EVEN');
    setFormDescription(rule.description || '');
    setFormIsActive(rule.is_active);
    setFormError(null);
    setIsModalOpen(true);
  };

  // Handle Form Submission
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formName.trim()) {
      setFormError('Rule name is required');
      return;
    }
    const mdr = parseFloat(formMDRRate);
    if (isNaN(mdr) || mdr < 0) {
      setFormError('MDR rate must be a valid non-negative percentage');
      return;
    }
    const gst = parseFloat(formGSTRate);
    if (isNaN(gst) || gst < 0) {
      setFormError('GST rate must be a valid non-negative percentage');
      return;
    }
    const tol = parseFloat(formFeeTolerance);
    if (isNaN(tol) || tol < 0) {
      setFormError('Fee tolerance must be a valid non-negative decimal');
      return;
    }

    setIsSubmitting(true);
    setFormError(null);

    const payload: Partial<ContractRule> = {
      name: formName.trim(),
      mdr_rate: formMDRRate.trim(),
      gst_rate: formGSTRate.trim(),
      fee_tolerance: formFeeTolerance.trim(),
      rounding_precision: formPrecision,
      rounding_mode: formRoundingMode,
      description: formDescription.trim(),
      is_active: formIsActive,
    };

    if (editingRule) {
      payload.id = editingRule.id;
    }

    const success = await saveContractRule(payload);
    setIsSubmitting(false);

    if (success) {
      setIsModalOpen(false);
    } else {
      setFormError('Failed to save contract rule. Please try again.');
    }
  };

  // Handle Activate
  const handleActivate = async (id: string) => {
    await activateContractRule(id);
  };

  // Handle Delete
  const handleDelete = async (rule: ContractRule) => {
    if (rule.is_active) {
      alert('Cannot delete the currently active contract rule. Please activate another rule first.');
      return;
    }
    if (window.confirm(`Are you sure you want to delete "${rule.name}"?`)) {
      await deleteContractRule(rule.id);
    }
  };

  // Helper calculation for preview
  const calculatePreview = (grossStr: string, mdrRateStr: string, gstRateStr: string, precision: number) => {
    const gross = parseFloat(grossStr) || 0;
    let mdr = parseFloat(mdrRateStr) || 0;
    if (mdr > 1) mdr = mdr / 100; // normalize percentage
    let gst = parseFloat(gstRateStr) || 0;
    if (gst > 1) gst = gst / 100;

    const mdrFee = gross * mdr;
    const gstTax = mdrFee * gst;
    const totalDeductions = mdrFee + gstTax;
    const netPayout = gross - totalDeductions;

    return {
      mdrFee: mdrFee.toFixed(precision),
      gstTax: gstTax.toFixed(precision),
      totalDeductions: totalDeductions.toFixed(precision),
      netPayout: netPayout.toFixed(precision),
    };
  };

  // Active preview numbers
  const activePreview = useMemo(() => {
    if (!activeRule) {
      return calculatePreview(simAmount, '2.0', '18.0', 4);
    }
    return calculatePreview(
      simAmount,
      activeRule.mdr_rate,
      activeRule.gst_rate,
      activeRule.rounding_precision || 4
    );
  }, [activeRule, simAmount]);

  // Modal preview numbers
  const modalPreview = useMemo(() => {
    return calculatePreview(simAmount, formMDRRate, formGSTRate, formPrecision);
  }, [simAmount, formMDRRate, formGSTRate, formPrecision]);

  const formatRate = (rateStr: string) => {
    const val = parseFloat(rateStr);
    if (isNaN(val)) return '0.00%';
    if (val < 1 && val > 0) {
      return `${(val * 100).toFixed(2)}%`;
    }
    return `${val.toFixed(2)}%`;
  };

  return (
    <div className="space-y-6">
      {/* Top Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-ledger-700/60 pb-5">
        <div>
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-lg bg-recon-indigo/10 border border-recon-indigo/30 text-recon-indigo">
              <Scale className="w-5 h-5" />
            </div>
            <div>
              <h1 className="text-xl font-bold tracking-tight text-white flex items-center gap-2">
                Contract Rules & Fee Engine
                <span className="text-xs font-mono px-2 py-0.5 rounded bg-recon-indigo/10 text-recon-indigo border border-recon-indigo/30">
                  DYNAMIC MDR / GST
                </span>
              </h1>
              <p className="text-xs text-ledger-400 mt-0.5">
                Configure agreed gateway fee tiers, statutory GST, fee variance tolerances, and rounding methods for real-time reconciliation.
              </p>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2.5">
          <button
            onClick={() => fetchContractRules()}
            disabled={isLoadingRules}
            className="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-ledger-800 hover:bg-ledger-700 text-ledger-300 hover:text-white border border-ledger-700 text-xs font-medium transition-all"
            title="Reload contract rules"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${isLoadingRules ? 'animate-spin' : ''}`} />
            Refresh
          </button>
          <button
            onClick={handleOpenCreate}
            className="flex items-center gap-2 px-3.5 py-2 rounded-lg bg-recon-indigo hover:bg-recon-indigo-hover text-white text-xs font-semibold shadow-sm transition-all"
          >
            <Plus className="w-4 h-4" />
            Add Contract Rule
          </button>
        </div>
      </div>

      {/* Active Rule Banner & Metric Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {/* Active Contract Status */}
        <div className="p-4 rounded-xl bg-ledger-900 border border-recon-emerald/30 relative overflow-hidden shadow-sm">
          <div className="flex items-center justify-between mb-2">
            <span className="text-[11px] font-mono uppercase tracking-wider text-recon-emerald flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-recon-emerald animate-pulse" />
              Active Contract
            </span>
            <ShieldCheck className="w-4 h-4 text-recon-emerald" />
          </div>
          <div className="text-base font-bold text-white truncate">
            {activeRule ? activeRule.name : 'Institutional Default'}
          </div>
          <div className="text-[11px] text-ledger-400 mt-1 line-clamp-1">
            {activeRule?.description || 'System standard baseline contract'}
          </div>
        </div>

        {/* Agreed MDR Rate */}
        <div className="p-4 rounded-xl bg-ledger-900 border border-ledger-800 shadow-sm">
          <div className="flex items-center justify-between mb-2">
            <span className="text-[11px] font-mono uppercase tracking-wider text-ledger-400 flex items-center gap-1.5">
              <Percent className="w-3.5 h-3.5 text-recon-indigo" />
              Agreed MDR Rate
            </span>
          </div>
          <div className="text-2xl font-mono font-bold text-white">
            {activeRule ? formatRate(activeRule.mdr_rate) : '2.00%'}
          </div>
          <div className="text-[11px] text-ledger-500 mt-1 font-mono">
            Gateway Transaction Fee
          </div>
        </div>

        {/* Statutory GST */}
        <div className="p-4 rounded-xl bg-ledger-900 border border-ledger-800 shadow-sm">
          <div className="flex items-center justify-between mb-2">
            <span className="text-[11px] font-mono uppercase tracking-wider text-ledger-400 flex items-center gap-1.5">
              <Layers className="w-3.5 h-3.5 text-recon-amber" />
              Statutory GST Rate
            </span>
          </div>
          <div className="text-2xl font-mono font-bold text-white">
            {activeRule ? formatRate(activeRule.gst_rate) : '18.00%'}
          </div>
          <div className="text-[11px] text-ledger-500 mt-1 font-mono">
            Applicable on PG fee
          </div>
        </div>

        {/* Fee Tolerance & Rounding */}
        <div className="p-4 rounded-xl bg-ledger-900 border border-ledger-800 shadow-sm">
          <div className="flex items-center justify-between mb-2">
            <span className="text-[11px] font-mono uppercase tracking-wider text-ledger-400 flex items-center gap-1.5">
              <Sliders className="w-3.5 h-3.5 text-recon-cyan" />
              Tolerance & Rounding
            </span>
          </div>
          <div className="text-lg font-mono font-bold text-white">
            ±₹{activeRule ? activeRule.fee_tolerance : '0.0100'}
          </div>
          <div className="text-[11px] text-ledger-400 mt-1 font-mono flex items-center gap-1.5">
            <span className="px-1.5 py-0.2 rounded bg-ledger-800 text-ledger-300 border border-ledger-700 text-[10px]">
              {activeRule ? activeRule.rounding_mode : 'HALF_EVEN'}
            </span>
            <span className="text-ledger-500 text-[10px]">
              {activeRule ? `${activeRule.rounding_precision} decimals` : '4 decimals'}
            </span>
          </div>
        </div>
      </div>

      {/* Live Financial Matching Calculator Preview */}
      <div className="p-4 rounded-xl bg-ledger-900/90 border border-ledger-800 shadow-sm space-y-3">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-ledger-800 pb-3">
          <div className="flex items-center gap-2">
            <Calculator className="w-4 h-4 text-recon-indigo" />
            <span className="text-xs font-semibold text-white tracking-wide uppercase font-mono">
              Live Settlement Formula Simulation (Active Contract)
            </span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-[11px] text-ledger-400 font-mono">Sample Order:</span>
            <div className="relative">
              <span className="absolute left-2.5 top-1.5 text-xs text-ledger-500 font-mono">₹</span>
              <input
                type="number"
                step="100"
                value={simAmount}
                onChange={(e) => setSimAmount(e.target.value)}
                className="w-32 pl-6 pr-2 py-1 text-xs font-mono bg-ledger-950 border border-ledger-700 rounded text-white focus:outline-none focus:border-recon-indigo"
                placeholder="10000.00"
              />
            </div>
          </div>
        </div>

        <div className="grid grid-cols-2 md:grid-cols-5 gap-3 pt-1">
          <div className="p-2.5 rounded bg-ledger-950/70 border border-ledger-800/80">
            <span className="text-[10px] uppercase font-mono text-ledger-500 block">Gross Amount</span>
            <span className="text-sm font-mono font-semibold text-white">₹{parseFloat(simAmount || '0').toLocaleString('en-IN', { minimumFractionDigits: 2 })}</span>
          </div>
          <div className="p-2.5 rounded bg-ledger-950/70 border border-ledger-800/80">
            <span className="text-[10px] uppercase font-mono text-ledger-500 block">
              Expected MDR ({activeRule ? formatRate(activeRule.mdr_rate) : '2.00%'})
            </span>
            <span className="text-sm font-mono font-semibold text-recon-indigo">₹{activePreview.mdrFee}</span>
          </div>
          <div className="p-2.5 rounded bg-ledger-950/70 border border-ledger-800/80">
            <span className="text-[10px] uppercase font-mono text-ledger-500 block">
              Expected GST ({activeRule ? formatRate(activeRule.gst_rate) : '18.00%'})
            </span>
            <span className="text-sm font-mono font-semibold text-recon-amber">₹{activePreview.gstTax}</span>
          </div>
          <div className="p-2.5 rounded bg-ledger-950/70 border border-ledger-800/80">
            <span className="text-[10px] uppercase font-mono text-ledger-500 block">Total Deductions</span>
            <span className="text-sm font-mono font-semibold text-rose-400">₹{activePreview.totalDeductions}</span>
          </div>
          <div className="p-2.5 rounded bg-recon-emerald/10 border border-recon-emerald/30 col-span-2 md:col-span-1">
            <span className="text-[10px] uppercase font-mono text-recon-emerald block">Expected Net Payout</span>
            <span className="text-sm font-mono font-bold text-white">₹{activePreview.netPayout}</span>
          </div>
        </div>
      </div>

      {/* Search and Table */}
      <div className="space-y-3">
        <div className="flex flex-col sm:flex-row items-center justify-between gap-3">
          <div className="relative w-full sm:w-80">
            <Search className="w-3.5 h-3.5 absolute left-3 top-3 text-ledger-500" />
            <input
              type="text"
              placeholder="Search contract rules by name, description..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-9 pr-3 py-2 text-xs font-mono bg-ledger-900 border border-ledger-700 rounded-lg text-white placeholder-ledger-500 focus:outline-none focus:border-recon-indigo"
            />
          </div>
          <div className="text-xs font-mono text-ledger-400">
            Showing <span className="text-white font-semibold">{filteredRules.length}</span> of {contractRules.length} contract profiles
          </div>
        </div>

        {/* Rules Table */}
        <div className="rounded-xl border border-ledger-800 bg-ledger-900 overflow-hidden shadow-sm">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-ledger-950/90 text-ledger-400 font-mono text-[11px] uppercase tracking-wider border-b border-ledger-800">
                <tr>
                  <th className="py-3 px-4">Contract Rule</th>
                  <th className="py-3 px-4">Status</th>
                  <th className="py-3 px-4 font-mono">Agreed MDR</th>
                  <th className="py-3 px-4 font-mono">Statutory GST</th>
                  <th className="py-3 px-4 font-mono">Fee Tolerance</th>
                  <th className="py-3 px-4">Precision & Mode</th>
                  <th className="py-3 px-4 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-ledger-800/80">
                {filteredRules.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="py-8 text-center text-ledger-500 font-mono text-xs">
                      No contract rules found matching your query.
                    </td>
                  </tr>
                ) : (
                  filteredRules.map((rule) => {
                    return (
                      <tr
                        key={rule.id}
                        className={`hover:bg-ledger-800/40 transition-colors ${
                          rule.is_active ? 'bg-recon-emerald/[0.03]' : ''
                        }`}
                      >
                        <td className="py-3.5 px-4">
                          <div className="font-semibold text-white flex items-center gap-2">
                            {rule.name}
                            {rule.is_active && (
                              <span className="w-1.5 h-1.5 rounded-full bg-recon-emerald inline-block" />
                            )}
                          </div>
                          {rule.description && (
                            <div className="text-[11px] text-ledger-400 mt-0.5 line-clamp-1 max-w-sm">
                              {rule.description}
                            </div>
                          )}
                        </td>

                        <td className="py-3.5 px-4 whitespace-nowrap">
                          {rule.is_active ? (
                            <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-mono font-medium bg-recon-emerald/15 text-recon-emerald border border-recon-emerald/30 shadow-xs">
                              <CheckCircle2 className="w-3 h-3" />
                              ACTIVE ENGINE
                            </span>
                          ) : (
                            <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-mono text-ledger-400 bg-ledger-800 border border-ledger-700">
                              INACTIVE
                            </span>
                          )}
                        </td>

                        <td className="py-3.5 px-4 font-mono font-medium text-white whitespace-nowrap">
                          {formatRate(rule.mdr_rate)}
                        </td>

                        <td className="py-3.5 px-4 font-mono font-medium text-white whitespace-nowrap">
                          {formatRate(rule.gst_rate)}
                        </td>

                        <td className="py-3.5 px-4 font-mono text-ledger-300 whitespace-nowrap">
                          ±₹{rule.fee_tolerance}
                        </td>

                        <td className="py-3.5 px-4 whitespace-nowrap font-mono text-[11px] text-ledger-300">
                          <div className="flex items-center gap-1.5">
                            <span className="px-1.5 py-0.5 rounded bg-ledger-950 border border-ledger-700 text-recon-cyan">
                              {rule.rounding_mode}
                            </span>
                            <span className="text-ledger-400">
                              {rule.rounding_precision} Decimals
                            </span>
                          </div>
                        </td>

                        <td className="py-3.5 px-4 text-right whitespace-nowrap">
                          <div className="flex items-center justify-end gap-2">
                            {!rule.is_active && (
                              <button
                                onClick={() => handleActivate(rule.id)}
                                className="flex items-center gap-1 px-2.5 py-1 rounded bg-recon-indigo/20 hover:bg-recon-indigo text-recon-indigo hover:text-white border border-recon-indigo/40 text-[11px] font-medium transition-all"
                                title="Activate this contract rule for matching engine"
                              >
                                <Zap className="w-3 h-3" />
                                Set Active
                              </button>
                            )}

                            <button
                              onClick={() => handleOpenEdit(rule)}
                              className="p-1.5 rounded hover:bg-ledger-800 text-ledger-400 hover:text-white transition-colors"
                              title="Edit contract rule parameters"
                            >
                              <Edit2 className="w-3.5 h-3.5" />
                            </button>

                            <button
                              onClick={() => handleDelete(rule)}
                              disabled={rule.is_active}
                              className={`p-1.5 rounded transition-colors ${
                                rule.is_active
                                  ? 'text-ledger-600 cursor-not-allowed opacity-50'
                                  : 'hover:bg-rose-500/10 text-ledger-400 hover:text-rose-400'
                              }`}
                              title={
                                rule.is_active
                                  ? 'Active rule cannot be deleted'
                                  : 'Delete this rule'
                              }
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      {/* Add / Edit Contract Rule Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm">
          <div className="relative w-full max-w-xl bg-ledger-900 border border-ledger-700 rounded-xl shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150">
            {/* Modal Header */}
            <div className="flex items-center justify-between px-6 py-4 border-b border-ledger-800 bg-ledger-950/70">
              <div className="flex items-center gap-2.5">
                <div className="p-2 rounded bg-recon-indigo/10 text-recon-indigo border border-recon-indigo/30">
                  <Scale className="w-4 h-4" />
                </div>
                <div>
                  <h3 className="text-sm font-bold text-white">
                    {editingRule ? 'Edit Contract Rule' : 'New Contract Rule'}
                  </h3>
                  <p className="text-[11px] text-ledger-400">
                    Define exact gateway MDR, statutory GST rate, and discrepancy thresholds
                  </p>
                </div>
              </div>
              <button
                onClick={() => setIsModalOpen(false)}
                className="p-1 rounded-md text-ledger-400 hover:text-white hover:bg-ledger-800 transition-colors"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            {/* Modal Form */}
            <form onSubmit={handleSubmit} className="p-6 space-y-4">
              {formError && (
                <div className="p-3 rounded-lg bg-rose-500/10 border border-rose-500/30 text-rose-400 text-xs flex items-center gap-2">
                  <AlertCircle className="w-4 h-4 shrink-0" />
                  <span>{formError}</span>
                </div>
              )}

              {/* Rule Name */}
              <div>
                <label className="block text-xs font-medium text-ledger-300 mb-1">
                  Rule / Contract Name <span className="text-rose-400">*</span>
                </label>
                <input
                  type="text"
                  required
                  value={formName}
                  onChange={(e) => setFormName(e.target.value)}
                  placeholder="e.g. Razorpay Enterprise Custom Tier 1"
                  className="w-full px-3 py-2 text-xs font-mono bg-ledger-950 border border-ledger-700 rounded-lg text-white focus:outline-none focus:border-recon-indigo"
                />
              </div>

              {/* Two Column Grid for MDR & GST */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                {/* Agreed MDR Rate */}
                <div>
                  <label className="block text-xs font-medium text-ledger-300 mb-1">
                    Agreed MDR Rate (%) <span className="text-rose-400">*</span>
                  </label>
                  <div className="relative">
                    <input
                      type="number"
                      step="0.0001"
                      min="0"
                      max="100"
                      required
                      value={formMDRRate}
                      onChange={(e) => setFormMDRRate(e.target.value)}
                      placeholder="2.0000"
                      className="w-full px-3 py-2 pr-7 text-xs font-mono bg-ledger-950 border border-ledger-700 rounded-lg text-white focus:outline-none focus:border-recon-indigo"
                    />
                    <span className="absolute right-2.5 top-2 text-xs text-ledger-500 font-mono">%</span>
                  </div>
                  <span className="text-[10px] text-ledger-500 mt-0.5 block">
                    e.g. 2.00% or 1.85%
                  </span>
                </div>

                {/* Statutory GST Rate */}
                <div>
                  <label className="block text-xs font-medium text-ledger-300 mb-1">
                    Statutory GST Rate (%) <span className="text-rose-400">*</span>
                  </label>
                  <div className="relative">
                    <input
                      type="number"
                      step="0.0001"
                      min="0"
                      max="100"
                      required
                      value={formGSTRate}
                      onChange={(e) => setFormGSTRate(e.target.value)}
                      placeholder="18.0000"
                      className="w-full px-3 py-2 pr-7 text-xs font-mono bg-ledger-950 border border-ledger-700 rounded-lg text-white focus:outline-none focus:border-recon-indigo"
                    />
                    <span className="absolute right-2.5 top-2 text-xs text-ledger-500 font-mono">%</span>
                  </div>
                  <span className="text-[10px] text-ledger-500 mt-0.5 block">
                    Applicable on PG fee (standard 18.00%)
                  </span>
                </div>
              </div>

              {/* Two Column Grid for Fee Tolerance & Precision */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                {/* Fee Variance Tolerance */}
                <div>
                  <label className="block text-xs font-medium text-ledger-300 mb-1">
                    Fee Variance Tolerance (₹) <span className="text-rose-400">*</span>
                  </label>
                  <div className="relative">
                    <input
                      type="number"
                      step="0.0001"
                      min="0"
                      required
                      value={formFeeTolerance}
                      onChange={(e) => setFormFeeTolerance(e.target.value)}
                      placeholder="0.0100"
                      className="w-full pl-6 pr-3 py-2 text-xs font-mono bg-ledger-950 border border-ledger-700 rounded-lg text-white focus:outline-none focus:border-recon-indigo"
                    />
                    <span className="absolute left-2.5 top-2 text-xs text-ledger-500 font-mono">₹</span>
                  </div>
                  <span className="text-[10px] text-ledger-500 mt-0.5 block">
                    Variance allowed before flagging discrepancy
                  </span>
                </div>

                {/* Rounding Precision */}
                <div>
                  <label className="block text-xs font-medium text-ledger-300 mb-1">
                    Rounding Precision
                  </label>
                  <select
                    value={formPrecision}
                    onChange={(e) => setFormPrecision(parseInt(e.target.value, 10))}
                    className="w-full px-3 py-2 text-xs font-mono bg-ledger-950 border border-ledger-700 rounded-lg text-white focus:outline-none focus:border-recon-indigo"
                  >
                    <option value={4}>4 Decimals (Exact Fintech Sub-Cent)</option>
                    <option value={2}>2 Decimals (Standard Currency ₹0.01)</option>
                  </select>
                </div>
              </div>

              {/* Rounding Mode */}
              <div>
                <label className="block text-xs font-medium text-ledger-300 mb-1">
                  Rounding Mode
                </label>
                <select
                  value={formRoundingMode}
                  onChange={(e) => setFormRoundingMode(e.target.value)}
                  className="w-full px-3 py-2 text-xs font-mono bg-ledger-950 border border-ledger-700 rounded-lg text-white focus:outline-none focus:border-recon-indigo"
                >
                  <option value="HALF_EVEN">HALF_EVEN — Banker's Rounding (IEEE 754 FinTech Standard)</option>
                  <option value="HALF_UP">HALF_UP — Standard Commercial Rounding (Round half up)</option>
                  <option value="FLOOR">FLOOR — Truncate down towards negative infinity</option>
                  <option value="CEIL">CEIL — Round up towards positive infinity</option>
                </select>
              </div>

              {/* Description */}
              <div>
                <label className="block text-xs font-medium text-ledger-300 mb-1">
                  Audit Notes / Description
                </label>
                <textarea
                  rows={2}
                  value={formDescription}
                  onChange={(e) => setFormDescription(e.target.value)}
                  placeholder="Notes about client SLA, contract amendment date, or payment gateway agreement..."
                  className="w-full px-3 py-2 text-xs font-sans bg-ledger-950 border border-ledger-700 rounded-lg text-white focus:outline-none focus:border-recon-indigo resize-none"
                />
              </div>

              {/* Set Active Checkbox */}
              <div className="flex items-center gap-2 pt-1">
                <input
                  type="checkbox"
                  id="formIsActive"
                  checked={formIsActive}
                  onChange={(e) => setFormIsActive(e.target.checked)}
                  className="w-4 h-4 rounded border-ledger-700 bg-ledger-950 text-recon-indigo focus:ring-0"
                />
                <label htmlFor="formIsActive" className="text-xs text-ledger-300 select-none cursor-pointer">
                  Set this rule as the <span className="text-recon-emerald font-semibold">Active Reconciliation Engine Rule</span> immediately
                </label>
              </div>

              {/* Real-time Calculation Simulation in Modal */}
              <div className="p-3 rounded-lg bg-ledger-950 border border-ledger-800 text-xs font-mono space-y-1.5">
                <div className="text-[11px] text-ledger-400 font-semibold uppercase flex items-center gap-1.5">
                  <Calculator className="w-3.5 h-3.5 text-recon-cyan" />
                  Formula Preview for ₹10,000 Order:
                </div>
                <div className="flex items-center justify-between text-ledger-300 text-[11px]">
                  <span>MDR Fee ({formMDRRate}%): <span className="text-white font-semibold">₹{modalPreview.mdrFee}</span></span>
                  <span>GST ({formGSTRate}%): <span className="text-white font-semibold">₹{modalPreview.gstTax}</span></span>
                  <span>Net Payout: <span className="text-recon-emerald font-semibold">₹{modalPreview.netPayout}</span></span>
                </div>
              </div>

              {/* Modal Actions */}
              <div className="flex items-center justify-end gap-3 pt-3 border-t border-ledger-800">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="px-4 py-2 rounded-lg text-xs font-medium text-ledger-300 hover:text-white hover:bg-ledger-800 transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isSubmitting}
                  className="flex items-center gap-2 px-4 py-2 rounded-lg bg-recon-indigo hover:bg-recon-indigo-hover text-white text-xs font-semibold shadow-sm transition-all"
                >
                  {isSubmitting ? (
                    <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                  ) : (
                    <Check className="w-3.5 h-3.5" />
                  )}
                  {editingRule ? 'Update Contract Rule' : 'Save & Register Rule'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
