'use client';

import {useCallback, useEffect, useRef, useState} from 'react';

import {validateRouteRule} from '@/lib/api';
import {RouteRuleValidationResult} from '@/lib/types';

import {routeRuleValidationPayload, RouteRuleFormValues, RouteRuleValidationPayload} from '../_lib/form';

type UseRouteRuleValidationArgs = {
  accessToken: string;
  activeTenantId: string | null;
  formValues: RouteRuleFormValues;
};

export function useRouteRuleValidation({accessToken, activeTenantId, formValues}: UseRouteRuleValidationArgs) {
  const [validationResult, setValidationResult] = useState<RouteRuleValidationResult | null>(null);
  const [validationPending, setValidationPending] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const scheduledValidationKeyRef = useRef('');
  const inFlightValidationKeyRef = useRef('');
  const completedValidationKeyRef = useRef('');

  const runValidation = useCallback(async (payload: RouteRuleValidationPayload, validationKey: string) => {
    if (completedValidationKeyRef.current === validationKey || inFlightValidationKeyRef.current === validationKey) {
      return;
    }
    scheduledValidationKeyRef.current = scheduledValidationKeyRef.current === validationKey ? '' : scheduledValidationKeyRef.current;
    inFlightValidationKeyRef.current = validationKey;
    setValidationPending(true);
    try {
      const result = await validateRouteRule(accessToken, payload);
      if (inFlightValidationKeyRef.current === validationKey) {
        setValidationResult(result);
      }
    } catch {
      if (inFlightValidationKeyRef.current === validationKey) {
        setValidationResult(null);
      }
    } finally {
      if (inFlightValidationKeyRef.current === validationKey) {
        completedValidationKeyRef.current = validationKey;
        inFlightValidationKeyRef.current = '';
        setValidationPending(false);
      }
    }
  }, [accessToken]);

  useEffect(() => {
    const payload = routeRuleValidationPayload(formValues);
    const validationKey = `${accessToken}:${activeTenantId}:${JSON.stringify(payload)}`;

    if (!accessToken || !payload.matchValue) {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
        debounceRef.current = null;
      }
      scheduledValidationKeyRef.current = '';
      inFlightValidationKeyRef.current = '';
      completedValidationKeyRef.current = '';
      setValidationResult(null);
      setValidationPending(false);
      return;
    }

    if (
      completedValidationKeyRef.current === validationKey ||
      inFlightValidationKeyRef.current === validationKey ||
      scheduledValidationKeyRef.current === validationKey
    ) {
      return;
    }

    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
      debounceRef.current = null;
    }

    scheduledValidationKeyRef.current = validationKey;
    debounceRef.current = setTimeout(() => {
      runValidation(payload, validationKey);
    }, 500);
    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
        debounceRef.current = null;
      }
    };
  }, [accessToken, activeTenantId, formValues, runValidation]);

  return {validationPending, validationResult};
}
