package keeper

import (
	"bytes"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
)

// return a specific delegation
func (k Keeper) GetDelegation(ctx sdk.Context,
	delAddr sdk.AccAddress, valAddr sdk.ValAddress) (
	delegation types.Delegation, found bool) {

	store := ctx.KVStore(k.storeKey)
	key := GetDelegationKey(delAddr, valAddr)
	value := store.Get(key)
	if value == nil {
		return delegation, false
	}

	delegation = types.MustUnmarshalDelegation(k.cdc, value)
	return delegation, true
}

// return all delegations used during genesis dump
func (k Keeper) GetAllDelegations(ctx sdk.Context) (delegations []types.Delegation) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, DelegationKey)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		delegation := types.MustUnmarshalDelegation(k.cdc, iterator.Value())
		delegations = append(delegations, delegation)
	}
	return delegations
}

// return all delegations to a specific validator. Useful for querier.
func (k Keeper) GetValidatorDelegations(ctx sdk.Context, valAddr sdk.ValAddress) (delegations []types.Delegation) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, DelegationKey)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		delegation := types.MustUnmarshalDelegation(k.cdc, iterator.Value())
		if delegation.GetValidatorAddr().Equals(valAddr) {
			delegations = append(delegations, delegation)
		}
	}
	return delegations
}

// return a given amount of all the delegations from a delegator
func (k Keeper) GetDelegatorDelegations(ctx sdk.Context, delegator sdk.AccAddress,
	maxRetrieve uint16) (delegations []types.Delegation) {

	delegations = make([]types.Delegation, maxRetrieve)

	store := ctx.KVStore(k.storeKey)
	delegatorPrefixKey := GetDelegationsKey(delegator)
	iterator := sdk.KVStorePrefixIterator(store, delegatorPrefixKey)
	defer iterator.Close()

	i := 0
	for ; iterator.Valid() && i < int(maxRetrieve); iterator.Next() {
		delegation := types.MustUnmarshalDelegation(k.cdc, iterator.Value())
		delegations[i] = delegation
		i++
	}
	return delegations[:i] // trim if the array length < maxRetrieve
}

// set the delegation
func (k Keeper) SetDelegation(ctx sdk.Context, delegation types.Delegation) {
	store := ctx.KVStore(k.storeKey)
	b := types.MustMarshalDelegation(k.cdc, delegation)
	store.Set(GetDelegationKey(delegation.DelegatorAddr, delegation.ValidatorAddr), b)
}

// remove a delegation from store
func (k Keeper) RemoveDelegation(ctx sdk.Context, delegation types.Delegation) {
	k.OnDelegationRemoved(ctx, delegation.DelegatorAddr, delegation.ValidatorAddr)
	store := ctx.KVStore(k.storeKey)
	store.Delete(GetDelegationKey(delegation.DelegatorAddr, delegation.ValidatorAddr))
}

//_____________________________________________________________________________________

// return a given amount of all the delegator unbonding-delegations
func (k Keeper) GetUnbondingDelegations(ctx sdk.Context, delegator sdk.AccAddress,
	maxRetrieve uint16) (unbondingDelegations []types.UnbondingDelegation) {

	unbondingDelegations = make([]types.UnbondingDelegation, maxRetrieve)

	store := ctx.KVStore(k.storeKey)
	delegatorPrefixKey := GetUBDsKey(delegator)
	iterator := sdk.KVStorePrefixIterator(store, delegatorPrefixKey)
	defer iterator.Close()

	i := 0
	for ; iterator.Valid() && i < int(maxRetrieve); iterator.Next() {
		unbondingDelegation := types.MustUnmarshalUBD(k.cdc, iterator.Value())
		unbondingDelegations[i] = unbondingDelegation
		i++
	}
	return unbondingDelegations[:i] // trim if the array length < maxRetrieve
}

// return a unbonding delegation
func (k Keeper) GetUnbondingDelegation(ctx sdk.Context,
	delAddr sdk.AccAddress, valAddr sdk.ValAddress) (ubd types.UnbondingDelegation, found bool) {

	store := ctx.KVStore(k.storeKey)
	key := GetUBDKey(delAddr, valAddr)
	value := store.Get(key)
	if value == nil {
		return ubd, false
	}

	ubd = types.MustUnmarshalUBD(k.cdc, value)
	return ubd, true
}

// return all unbonding delegations from a particular validator
func (k Keeper) GetUnbondingDelegationsFromValidator(ctx sdk.Context, valAddr sdk.ValAddress) (ubds []types.UnbondingDelegation) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, GetUBDsByValIndexKey(valAddr))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := GetUBDKeyFromValIndexKey(iterator.Key())
		value := store.Get(key)
		ubd := types.MustUnmarshalUBD(k.cdc, value)
		ubds = append(ubds, ubd)
	}
	return ubds
}

// iterate through all of the unbonding delegations
func (k Keeper) IterateUnbondingDelegations(ctx sdk.Context, fn func(index int64, ubd types.UnbondingDelegation) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, UnbondingDelegationKey)
	defer iterator.Close()

	for i := int64(0); iterator.Valid(); iterator.Next() {
		ubd := types.MustUnmarshalUBD(k.cdc, iterator.Value())
		if stop := fn(i, ubd); stop {
			break
		}
		i++
	}
}

// set the unbonding delegation and associated index
func (k Keeper) SetUnbondingDelegation(ctx sdk.Context, ubd types.UnbondingDelegation) {
	store := ctx.KVStore(k.storeKey)
	bz := types.MustMarshalUBD(k.cdc, ubd)
	key := GetUBDKey(ubd.DelegatorAddr, ubd.ValidatorAddr)
	store.Set(key, bz)
	store.Set(GetUBDByValIndexKey(ubd.DelegatorAddr, ubd.ValidatorAddr), []byte{}) // index, store empty bytes
}

// remove the unbonding delegation object and associated index
func (k Keeper) RemoveUnbondingDelegation(ctx sdk.Context, ubd types.UnbondingDelegation) {
	store := ctx.KVStore(k.storeKey)
	key := GetUBDKey(ubd.DelegatorAddr, ubd.ValidatorAddr)
	store.Delete(key)
	store.Delete(GetUBDByValIndexKey(ubd.DelegatorAddr, ubd.ValidatorAddr))
}

//________________________________________________
// unbonding delegation queue timeslice operations

// gets a specific unbonding queue timeslice. A timeslice is a slice of DVPairs
// corresponding to unbonding delegations that expire at a certain time.
func (k Keeper) GetUBDQueueTimeSlice(ctx sdk.Context, timestamp time.Time) (dvPairs []types.DVPair) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(GetUnbondingDelegationTimeKey(timestamp))
	if bz == nil {
		return []types.DVPair{}
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(bz, &dvPairs)
	return dvPairs
}

// Sets a specific unbonding queue timeslice.
func (k Keeper) SetUBDQueueTimeSlice(ctx sdk.Context, timestamp time.Time, keys []types.DVPair) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshalBinaryLengthPrefixed(keys)
	store.Set(GetUnbondingDelegationTimeKey(timestamp), bz)
}

// Insert an unbonding delegation to the appropriate timeslice in the unbonding queue
func (k Keeper) InsertUBDQueue(ctx sdk.Context, ubd types.UnbondingDelegation) {
	timeSlice := k.GetUBDQueueTimeSlice(ctx, ubd.MinTime)
	dvPair := types.DVPair{ubd.DelegatorAddr, ubd.ValidatorAddr}
	if len(timeSlice) == 0 {
		k.SetUBDQueueTimeSlice(ctx, ubd.MinTime, []types.DVPair{dvPair})
	} else {
		timeSlice = append(timeSlice, dvPair)
		k.SetUBDQueueTimeSlice(ctx, ubd.MinTime, timeSlice)
	}
}

// Returns all the unbonding queue timeslices from time 0 until endTime
func (k Keeper) UBDQueueIterator(ctx sdk.Context, endTime time.Time) sdk.Iterator {
	store := ctx.KVStore(k.storeKey)
	return store.Iterator(UnbondingQueueKey,
		sdk.InclusiveEndBytes(GetUnbondingDelegationTimeKey(endTime)))
}

// Returns a concatenated list of all the timeslices before currTime, and deletes the timeslices from the queue
func (k Keeper) DequeueAllMatureUBDQueue(ctx sdk.Context,
	currTime time.Time) (matureUnbonds []types.DVPair) {

	store := ctx.KVStore(k.storeKey)
	// gets an iterator for all timeslices from time 0 until the current Blockheader time
	unbondingTimesliceIterator := k.UBDQueueIterator(ctx, ctx.BlockHeader().Time)
	for ; unbondingTimesliceIterator.Valid(); unbondingTimesliceIterator.Next() {
		timeslice := []types.DVPair{}
		value := unbondingTimesliceIterator.Value()
		k.cdc.MustUnmarshalBinaryLengthPrefixed(value, &timeslice)
		matureUnbonds = append(matureUnbonds, timeslice...)
		store.Delete(unbondingTimesliceIterator.Key())
	}
	return matureUnbonds
}

//_____________________________________________________________________________________

// return a given amount of all the delegator redelegations
func (k Keeper) GetRedelegations(ctx sdk.Context, delegator sdk.AccAddress,
	maxRetrieve uint16) (redelegations []types.Redelegation) {
	redelegations = make([]types.Redelegation, maxRetrieve)

	store := ctx.KVStore(k.storeKey)
	delegatorPrefixKey := GetREDsKey(delegator)
	iterator := sdk.KVStorePrefixIterator(store, delegatorPrefixKey)
	defer iterator.Close()

	i := 0
	for ; iterator.Valid() && i < int(maxRetrieve); iterator.Next() {
		redelegation := types.MustUnmarshalRED(k.cdc, iterator.Value())
		redelegations[i] = redelegation
		i++
	}
	return redelegations[:i] // trim if the array length < maxRetrieve
}

// return a redelegation
func (k Keeper) GetRedelegation(ctx sdk.Context,
	delAddr sdk.AccAddress, valSrcAddr, valDstAddr sdk.ValAddress) (red types.Redelegation, found bool) {

	store := ctx.KVStore(k.storeKey)
	key := GetREDKey(delAddr, valSrcAddr, valDstAddr)
	value := store.Get(key)
	if value == nil {
		return red, false
	}

	red = types.MustUnmarshalRED(k.cdc, value)
	return red, true
}

// return all redelegations from a particular validator
func (k Keeper) GetRedelegationsFromValidator(ctx sdk.Context, valAddr sdk.ValAddress) (reds []types.Redelegation) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, GetREDsFromValSrcIndexKey(valAddr))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := GetREDKeyFromValSrcIndexKey(iterator.Key())
		value := store.Get(key)
		red := types.MustUnmarshalRED(k.cdc, value)
		reds = append(reds, red)
	}
	return reds
}

// check if validator is receiving a redelegation
func (k Keeper) HasReceivingRedelegation(ctx sdk.Context,
	delAddr sdk.AccAddress, valDstAddr sdk.ValAddress) bool {

	store := ctx.KVStore(k.storeKey)
	prefix := GetREDsByDelToValDstIndexKey(delAddr, valDstAddr)
	iterator := sdk.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	found := false
	if iterator.Valid() {
		found = true
	}
	return found
}

// set a redelegation and associated index
func (k Keeper) SetRedelegation(ctx sdk.Context, red types.Redelegation) {
	store := ctx.KVStore(k.storeKey)
	bz := types.MustMarshalRED(k.cdc, red)
	key := GetREDKey(red.DelegatorAddr, red.ValidatorSrcAddr, red.ValidatorDstAddr)
	store.Set(key, bz)
	store.Set(GetREDByValSrcIndexKey(red.DelegatorAddr, red.ValidatorSrcAddr, red.ValidatorDstAddr), []byte{})
	store.Set(GetREDByValDstIndexKey(red.DelegatorAddr, red.ValidatorSrcAddr, red.ValidatorDstAddr), []byte{})
}

// iterate through all redelegations
func (k Keeper) IterateRedelegations(ctx sdk.Context, fn func(index int64, red types.Redelegation) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, RedelegationKey)
	defer iterator.Close()

	for i := int64(0); iterator.Valid(); iterator.Next() {
		red := types.MustUnmarshalRED(k.cdc, iterator.Value())
		if stop := fn(i, red); stop {
			break
		}
		i++
	}
}

// remove a redelegation object and associated index
func (k Keeper) RemoveRedelegation(ctx sdk.Context, red types.Redelegation) {
	store := ctx.KVStore(k.storeKey)
	redKey := GetREDKey(red.DelegatorAddr, red.ValidatorSrcAddr, red.ValidatorDstAddr)
	store.Delete(redKey)
	store.Delete(GetREDByValSrcIndexKey(red.DelegatorAddr, red.ValidatorSrcAddr, red.ValidatorDstAddr))
	store.Delete(GetREDByValDstIndexKey(red.DelegatorAddr, red.ValidatorSrcAddr, red.ValidatorDstAddr))
}

//________________________________________________
// redelegation queue timeslice operations

// Gets a specific redelegation queue timeslice. A timeslice is a slice of DVVTriplets corresponding to redelegations
// that expire at a certain time.
func (k Keeper) GetRedelegationQueueTimeSlice(ctx sdk.Context, timestamp time.Time) (dvvTriplets []types.DVVTriplet) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(GetRedelegationTimeKey(timestamp))
	if bz == nil {
		return []types.DVVTriplet{}
	}
	k.cdc.MustUnmarshalBinaryLengthPrefixed(bz, &dvvTriplets)
	return dvvTriplets
}

// Sets a specific redelegation queue timeslice.
func (k Keeper) SetRedelegationQueueTimeSlice(ctx sdk.Context, timestamp time.Time, keys []types.DVVTriplet) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshalBinaryLengthPrefixed(keys)
	store.Set(GetRedelegationTimeKey(timestamp), bz)
}

// Insert an redelegation delegation to the appropriate timeslice in the redelegation queue
func (k Keeper) InsertRedelegationQueue(ctx sdk.Context, red types.Redelegation) {
	timeSlice := k.GetRedelegationQueueTimeSlice(ctx, red.MinTime)
	dvvTriplet := types.DVVTriplet{red.DelegatorAddr, red.ValidatorSrcAddr, red.ValidatorDstAddr}
	if len(timeSlice) == 0 {
		k.SetRedelegationQueueTimeSlice(ctx, red.MinTime, []types.DVVTriplet{dvvTriplet})
	} else {
		timeSlice = append(timeSlice, dvvTriplet)
		k.SetRedelegationQueueTimeSlice(ctx, red.MinTime, timeSlice)
	}
}

// Returns all the redelegation queue timeslices from time 0 until endTime
func (k Keeper) RedelegationQueueIterator(ctx sdk.Context, endTime time.Time) sdk.Iterator {
	store := ctx.KVStore(k.storeKey)
	return store.Iterator(RedelegationQueueKey, sdk.InclusiveEndBytes(GetRedelegationTimeKey(endTime)))
}

// Returns a concatenated list of all the timeslices before currTime, and deletes the timeslices from the queue
func (k Keeper) DequeueAllMatureRedelegationQueue(ctx sdk.Context, currTime time.Time) (matureRedelegations []types.DVVTriplet) {
	store := ctx.KVStore(k.storeKey)
	// gets an iterator for all timeslices from time 0 until the current Blockheader time
	redelegationTimesliceIterator := k.RedelegationQueueIterator(ctx, ctx.BlockHeader().Time)
	for ; redelegationTimesliceIterator.Valid(); redelegationTimesliceIterator.Next() {
		timeslice := []types.DVVTriplet{}
		value := redelegationTimesliceIterator.Value()
		k.cdc.MustUnmarshalBinaryLengthPrefixed(value, &timeslice)
		matureRedelegations = append(matureRedelegations, timeslice...)
		store.Delete(redelegationTimesliceIterator.Key())
	}
	return matureRedelegations
}

//_____________________________________________________________________________________

// Perform a delegation, set/update everything necessary within the store.
func (k Keeper) Delegate(ctx sdk.Context, delAddr sdk.AccAddress, bondAmt sdk.Coin,
	validator types.Validator, subtractAccount bool) (newShares sdk.Dec, err sdk.Error) {

	// In some situations, the exchange rate becomes invalid, e.g. if
	// validator loses all tokens due to slashing.  In this case,
	// make all future delegations invalid.
	if validator.DelegatorShareExRate().IsZero() {
		return sdk.ZeroDec(), types.ErrDelegatorShareExRateInvalid(k.Codespace())
	}

	// Get or create the delegator delegation
	delegation, found := k.GetDelegation(ctx, delAddr, validator.OperatorAddr)
	if !found {
		delegation = types.Delegation{
			DelegatorAddr: delAddr,
			ValidatorAddr: validator.OperatorAddr,
			Shares:        sdk.ZeroDec(),
		}
	}

	// call the appropriate hook if present
	if found {
		k.OnDelegationSharesModified(ctx, delAddr, validator.OperatorAddr)
	} else {
		k.OnDelegationCreated(ctx, delAddr, validator.OperatorAddr)
	}

	if subtractAccount {
		// Account new shares, save
		_, _, err = k.bankKeeper.SubtractCoins(ctx, delegation.DelegatorAddr, sdk.Coins{bondAmt})
		if err != nil {
			return
		}
	}

	validator, newShares = k.AddValidatorTokensAndShares(ctx, validator, bondAmt.Amount)

	// Update delegation
	delegation.Shares = delegation.Shares.Add(newShares)
	k.SetDelegation(ctx, delegation)

	return newShares, nil
}

// unbond the the delegation return
func (k Keeper) unbond(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress,
	shares sdk.Dec) (amount sdk.Int, err sdk.Error) {

	// check if delegation has any shares in it unbond
	delegation, found := k.GetDelegation(ctx, delAddr, valAddr)
	if !found {
		return amount, types.ErrNoDelegatorForAddress(k.Codespace())
	}

	k.OnDelegationSharesModified(ctx, delAddr, valAddr)

	// retrieve the amount to remove
	if delegation.Shares.LT(shares) {
		return amount, types.ErrNotEnoughDelegationShares(k.Codespace(), delegation.Shares.String())
	}

	// get validator
	validator, found := k.GetValidator(ctx, valAddr)
	if !found {
		return amount, types.ErrNoValidatorFound(k.Codespace())
	}

	// subtract shares from delegator
	delegation.Shares = delegation.Shares.Sub(shares)

	// remove the delegation
	if delegation.Shares.IsZero() {

		// if the delegation is the operator of the validator then
		// trigger a jail validator
		if bytes.Equal(delegation.DelegatorAddr, validator.OperatorAddr) && !validator.Jailed {
			k.jailValidator(ctx, validator)
			validator = k.mustGetValidator(ctx, validator.OperatorAddr)
		}

		k.RemoveDelegation(ctx, delegation)
	} else {
		// update the delegation
		k.SetDelegation(ctx, delegation)
	}

	// remove the coins from the validator
	validator, amount = k.RemoveValidatorTokensAndShares(ctx, validator, shares)

	if validator.DelegatorShares.IsZero() && validator.Status == sdk.Unbonded {
		// if not unbonded, we must instead remove validator in EndBlocker once it finishes its unbonding period
		k.RemoveValidator(ctx, validator.OperatorAddr)
	}

	return amount, nil
}

//______________________________________________________________________________________________________

// get info for begin functions: MinTime and CreationHeight
func (k Keeper) getBeginInfo(ctx sdk.Context, valSrcAddr sdk.ValAddress) (
	minTime time.Time, height int64, completeNow bool) {

	validator, found := k.GetValidator(ctx, valSrcAddr)

	switch {
	case !found || validator.Status == sdk.Bonded:

		// the longest wait - just unbonding period from now
		minTime = ctx.BlockHeader().Time.Add(k.UnbondingTime(ctx))
		height = ctx.BlockHeight()
		return minTime, height, false

	case validator.Status == sdk.Unbonded:
		return minTime, height, true

	case validator.Status == sdk.Unbonding:
		minTime = validator.UnbondingMinTime
		height = validator.UnbondingHeight
		return minTime, height, false

	default:
		panic("unknown validator status")
	}
}

// begin unbonding an unbonding record
func (k Keeper) BeginUnbonding(ctx sdk.Context, delAddr sdk.AccAddress,
	valAddr sdk.ValAddress, sharesAmount sdk.Dec) (types.UnbondingDelegation, sdk.Error) {

	// XXX remove this quickfix before merge
	// TODO quick fix, instead we should use an index, see https://github.com/cosmos/cosmos-sdk/issues/1402
	_, found := k.GetUnbondingDelegation(ctx, delAddr, valAddr)
	if found {
		return types.UnbondingDelegation{}, types.ErrExistingUnbondingDelegation(k.Codespace())
	}

	// create the unbonding delegation
	minTime, height, completeNow := k.getBeginInfo(ctx, valAddr)

	returnAmount, err := k.unbond(ctx, delAddr, valAddr, sharesAmount)
	if err != nil {
		return types.UnbondingDelegation{}, err
	}
	balance := sdk.NewCoin(k.BondDenom(ctx), returnAmount)

	// no need to create the ubd object just complete now
	if completeNow {
		_, _, err := k.bankKeeper.AddCoins(ctx, delAddr, sdk.Coins{balance})
		if err != nil {
			return types.UnbondingDelegation{}, err
		}
		return types.UnbondingDelegation{MinTime: minTime}, nil
	}

	ubd := NewUnbondingDelegation(delAddr, valAddr, height, minTime, balance)
	k.SetUnbondingDelegation(ctx, ubd)
	k.InsertUBDQueue(ctx, ubd)

	return ubd, nil
}

// complete unbonding an unbonding record
// CONTRACT: Expects unbonding passed in has finished the unbonding period
func (k Keeper) CompleteUnbonding(ctx sdk.Context, delAddr sdk.AccAddress,
	valAddr sdk.ValAddress, currentTime time.Time) sdk.Error {

	ubd, found := k.GetUnbondingDelegation(ctx, delAddr, valAddr)
	if !found {
		return types.ErrNoUnbondingDelegation(k.Codespace())
	}

	_, _, err := k.bankKeeper.AddCoins(ctx, ubd.DelegatorAddr, sdk.Coins{ubd.Balance})
	if err != nil {
		return err
	}
	k.RemoveUnbondingDelegation(ctx, ubd)
	return nil
}

// begin unbonding / redelegation; create a redelegation record
func (k Keeper) BeginRedelegation(ctx sdk.Context, delAddr sdk.AccAddress,
	valSrcAddr, valDstAddr sdk.ValAddress, sharesAmount sdk.Dec) (types.Redelegation, sdk.Error) {

	if bytes.Equal(valSrcAddr, valDstAddr) {
		return types.Redelegation{}, types.ErrSelfRedelegation(k.Codespace())
	}

	// check if there is already a redelgation in progress from src to dst
	// TODO quick fix, instead we should use an index, see https://github.com/cosmos/cosmos-sdk/issues/1402
	_, found := k.GetRedelegation(ctx, delAddr, valSrcAddr, valDstAddr)
	if found {
		return types.Redelegation{}, types.ErrConflictingRedelegation(k.Codespace())
	}

	// check if this is a transitive redelegation
	if k.HasReceivingRedelegation(ctx, delAddr, valSrcAddr) {
		return types.Redelegation{}, types.ErrTransitiveRedelegation(k.Codespace())
	}

	returnAmount, err := k.unbond(ctx, delAddr, valSrcAddr, sharesAmount)
	if err != nil {
		return types.Redelegation{}, err
	}

	if returnAmount.IsZero() {
		return types.Redelegation{}, types.ErrVerySmallRedelegation(k.Codespace())
	}
	returnCoin := sdk.NewCoin(k.BondDenom(ctx), returnAmount)

	dstValidator, found := k.GetValidator(ctx, valDstAddr)
	if !found {
		return types.Redelegation{}, types.ErrBadRedelegationDst(k.Codespace())
	}

	sharesCreated, err := k.Delegate(ctx, delAddr, returnCoin, dstValidator, false)
	if err != nil {
		return types.Redelegation{}, err
	}

	// create the unbonding delegation
	minTime, height, completeNow := k.getBeginInfo(ctx, valSrcAddr)

	if completeNow { // no need to create the redelegation object
		return types.Redelegation{MinTime: minTime}, nil
	}

	red := NewRedelegation(delAddr, valSrcAddr, valDstAddr, height, minTime,
		returnCoin, sharesAmount, sharesCreated)
	k.SetRedelegation(ctx, red)
	k.InsertRedelegationQueue(ctx, red)
	return red, nil
}

// complete unbonding an ongoing redelegation
func (k Keeper) CompleteRedelegation(ctx sdk.Context, delAddr sdk.AccAddress,
	valSrcAddr, valDstAddr sdk.ValAddress) sdk.Error {

	red, found := k.GetRedelegation(ctx, delAddr, valSrcAddr, valDstAddr)
	if !found {
		return types.ErrNoRedelegation(k.Codespace())
	}

	// ensure that enough time has passed
	ctxTime := ctx.BlockHeader().Time
	if red.MinTime.After(ctxTime) {
		return types.ErrNotMature(k.Codespace(), "redelegation", "unit-time", red.MinTime, ctxTime)
	}

	k.RemoveRedelegation(ctx, red)
	return nil
}
