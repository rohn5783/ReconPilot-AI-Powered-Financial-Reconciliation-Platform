import React, { useState, useEffect } from 'react';
import {
  X,
  Lock,
  Mail,
  User,
  Building,
  Shield,
  Eye,
  EyeOff,
  AlertCircle,
  ArrowRight,
  CheckCircle2,
  Sparkles,
} from 'lucide-react';
import { useReconStore } from '../../store/useReconStore';

export const AuthModal: React.FC = () => {
  const {
    isAuthModalOpen,
    authModalTab,
    authError,
    isAuthenticating,
    closeAuthModal,
    openAuthModal,
    login,
    register,
  } = useReconStore();

  const [tab, setTab] = useState<'login' | 'register'>('login');
  const [showPassword, setShowPassword] = useState(false);

  // Form states
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [fullName, setFullName] = useState('');
  const [companyName, setCompanyName] = useState('');
  const [role, setRole] = useState('FINANCE_ANALYST');
  const [formValidationErr, setFormValidationErr] = useState<string | null>(null);

  useEffect(() => {
    if (authModalTab) {
      setTab(authModalTab);
    }
  }, [authModalTab]);

  useEffect(() => {
    if (isAuthModalOpen) {
      setFormValidationErr(null);
    }
  }, [isAuthModalOpen, tab]);

  if (!isAuthModalOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormValidationErr(null);

    const cleanEmail = email.trim().toLowerCase();
    if (!cleanEmail || !cleanEmail.includes('@')) {
      setFormValidationErr('Please enter a valid business email address.');
      return;
    }

    if (!password || password.length < 6) {
      setFormValidationErr('Password must be at least 6 characters in length.');
      return;
    }

    if (tab === 'login') {
      await login({ email: cleanEmail, password });
    } else {
      if (!fullName.trim()) {
        setFormValidationErr('Full name is required for institutional audit logging.');
        return;
      }
      await register({
        email: cleanEmail,
        password,
        full_name: fullName.trim(),
        company_name: companyName.trim() || 'Institutional Ledger Corp',
        role,
      });
    }
  };

  const fillDemoAccount = () => {
    setEmail('analyst@reconpilot.io');
    setPassword('ReconPilot2026!');
    setFullName('Elena Rostova');
    setCompanyName('Stripe Sigma & Ramp Treasury');
    setRole('LEAD_FINANCE_ARCHITECT');
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm animate-fade-in">
      <div
        className="w-full max-w-md bg-ledger-900 border border-ledger-700/80 rounded-xl shadow-2xl overflow-hidden text-slate-100 flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="px-6 pt-6 pb-4 border-b border-ledger-800 bg-ledger-950/60 flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="w-8 h-8 rounded-lg bg-indigo-500/15 border border-indigo-500/30 flex items-center justify-center text-indigo-400">
              <Shield className="w-4 h-4" />
            </div>
            <div>
              <h2 className="text-sm font-semibold tracking-wide text-slate-100 uppercase font-mono">
                {tab === 'login' ? 'Operator Authentication' : 'Create Platform Account'}
              </h2>
              <p className="text-xs text-slate-400">
                {tab === 'login'
                  ? 'Access secure 3-way financial reconciliation engine'
                  : 'Enroll new operator into ledger audit system'}
              </p>
            </div>
          </div>
          <button
            onClick={closeAuthModal}
            className="p-1 rounded-md text-slate-400 hover:text-slate-200 hover:bg-ledger-800 transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Tab Switcher */}
        <div className="flex border-b border-ledger-800 bg-ledger-950/30">
          <button
            type="button"
            onClick={() => {
              setTab('login');
              openAuthModal('login');
            }}
            className={`flex-1 py-2.5 text-xs font-medium font-mono text-center border-b-2 transition-all ${
              tab === 'login'
                ? 'border-indigo-500 text-indigo-400 bg-indigo-500/5'
                : 'border-transparent text-slate-400 hover:text-slate-200'
            }`}
          >
            Sign In
          </button>
          <button
            type="button"
            onClick={() => {
              setTab('register');
              openAuthModal('register');
            }}
            className={`flex-1 py-2.5 text-xs font-medium font-mono text-center border-b-2 transition-all ${
              tab === 'register'
                ? 'border-indigo-500 text-indigo-400 bg-indigo-500/5'
                : 'border-transparent text-slate-400 hover:text-slate-200'
            }`}
          >
            Create Account
          </button>
        </div>

        {/* Error Alert */}
        {(formValidationErr || authError) && (
          <div className="mx-6 mt-4 p-3 rounded-lg bg-rose-500/10 border border-rose-500/30 flex items-start space-x-2 text-rose-300 text-xs">
            <AlertCircle className="w-4 h-4 text-rose-400 mt-0.5 flex-shrink-0" />
            <div className="flex-1 font-mono">{formValidationErr || authError}</div>
          </div>
        )}

        {/* Form Body */}
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {tab === 'register' && (
            <>
              <div>
                <label className="block text-[11px] font-mono uppercase tracking-wider text-slate-400 mb-1.5">
                  Full Name
                </label>
                <div className="relative">
                  <User className="w-4 h-4 text-slate-500 absolute left-3 top-2.5" />
                  <input
                    type="text"
                    required
                    value={fullName}
                    onChange={(e) => setFullName(e.target.value)}
                    placeholder="e.g. Elena Rostova"
                    className="w-full bg-ledger-950 border border-ledger-700 rounded-lg pl-9 pr-3 py-2 text-xs font-mono text-slate-100 placeholder-slate-500 focus:outline-none focus:border-indigo-500 transition-colors"
                  />
                </div>
              </div>

              <div>
                <label className="block text-[11px] font-mono uppercase tracking-wider text-slate-400 mb-1.5">
                  Company / Organization
                </label>
                <div className="relative">
                  <Building className="w-4 h-4 text-slate-500 absolute left-3 top-2.5" />
                  <input
                    type="text"
                    value={companyName}
                    onChange={(e) => setCompanyName(e.target.value)}
                    placeholder="e.g. Stripe Treasury / Ramp"
                    className="w-full bg-ledger-950 border border-ledger-700 rounded-lg pl-9 pr-3 py-2 text-xs font-mono text-slate-100 placeholder-slate-500 focus:outline-none focus:border-indigo-500 transition-colors"
                  />
                </div>
              </div>

              <div>
                <label className="block text-[11px] font-mono uppercase tracking-wider text-slate-400 mb-1.5">
                  Operational Role
                </label>
                <select
                  value={role}
                  onChange={(e) => setRole(e.target.value)}
                  className="w-full bg-ledger-950 border border-ledger-700 rounded-lg px-3 py-2 text-xs font-mono text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors"
                >
                  <option value="FINANCE_ANALYST">Finance Operations Analyst</option>
                  <option value="LEAD_FINANCE_ARCHITECT">Lead Financial Architect</option>
                  <option value="COMPLIANCE_AUDITOR">Compliance & Risk Auditor</option>
                  <option value="CFO_VP_FINANCE">VP Finance / Controller</option>
                </select>
              </div>
            </>
          )}

          <div>
            <label className="block text-[11px] font-mono uppercase tracking-wider text-slate-400 mb-1.5">
              Email Address
            </label>
            <div className="relative">
              <Mail className="w-4 h-4 text-slate-500 absolute left-3 top-2.5" />
              <input
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="operator@company.com"
                className="w-full bg-ledger-950 border border-ledger-700 rounded-lg pl-9 pr-3 py-2 text-xs font-mono text-slate-100 placeholder-slate-500 focus:outline-none focus:border-indigo-500 transition-colors"
              />
            </div>
          </div>

          <div>
            <label className="block text-[11px] font-mono uppercase tracking-wider text-slate-400 mb-1.5">
              Password
            </label>
            <div className="relative">
              <Lock className="w-4 h-4 text-slate-500 absolute left-3 top-2.5" />
              <input
                type={showPassword ? 'text' : 'password'}
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••••••"
                className="w-full bg-ledger-950 border border-ledger-700 rounded-lg pl-9 pr-9 py-2 text-xs font-mono text-slate-100 placeholder-slate-500 focus:outline-none focus:border-indigo-500 transition-colors"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3 top-2.5 text-slate-500 hover:text-slate-300"
              >
                {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
            {tab === 'register' && (
              <p className="mt-1 text-[10px] text-slate-500">Minimum 6 characters. Hashed with bcrypt salt.</p>
            )}
          </div>

          {/* Quick Prefill helper */}
          <div className="flex items-center justify-between pt-1">
            <button
              type="button"
              onClick={fillDemoAccount}
              className="text-[11px] font-mono text-indigo-400 hover:text-indigo-300 flex items-center space-x-1"
            >
              <Sparkles className="w-3 h-3" />
              <span>Fill demo operator</span>
            </button>
          </div>

          {/* Submit Button */}
          <button
            type="submit"
            disabled={isAuthenticating}
            className="w-full mt-2 py-2.5 px-4 bg-indigo-600 hover:bg-indigo-500 disabled:bg-indigo-800 disabled:opacity-60 text-white font-mono text-xs font-semibold rounded-lg shadow-lg flex items-center justify-center space-x-2 transition-all"
          >
            {isAuthenticating ? (
              <span className="flex items-center space-x-2">
                <svg className="animate-spin h-4 w-4 text-white" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
                </svg>
                <span>Authenticating Credentials...</span>
              </span>
            ) : (
              <span className="flex items-center space-x-1.5">
                <span>{tab === 'login' ? 'Authorize Session' : 'Create Platform Profile'}</span>
                <ArrowRight className="w-3.5 h-3.5" />
              </span>
            )}
          </button>
        </form>

        {/* Footer */}
        <div className="px-6 py-3 bg-ledger-950/80 border-t border-ledger-800 text-center">
          <p className="text-xs text-slate-400">
            {tab === 'login' ? (
              <>
                Need a new reconciliation operator seat?{' '}
                <button
                  type="button"
                  onClick={() => {
                    setTab('register');
                    openAuthModal('register');
                  }}
                  className="text-indigo-400 hover:text-indigo-300 font-medium underline underline-offset-2"
                >
                  Create Account
                </button>
              </>
            ) : (
              <>
                Already have credentials?{' '}
                <button
                  type="button"
                  onClick={() => {
                    setTab('login');
                    openAuthModal('login');
                  }}
                  className="text-indigo-400 hover:text-indigo-300 font-medium underline underline-offset-2"
                >
                  Sign In
                </button>
              </>
            )}
          </p>
        </div>
      </div>
    </div>
  );
};
