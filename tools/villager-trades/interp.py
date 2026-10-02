#!/usr/bin/env python3
"""Reads VillagerTrades from an obfuscated Minecraft client jar.

Interprets the javap -c output of VillagerTrades' static initialiser (or the
lambda that fills TRADES) with a tiny stack machine and prints the trades.
See README.md in this folder for the steps (finding the obfuscated classes,
building the Items/Blocks maps). Used to check the built-in trades of
app/internal/discovery/vanilla/trades.go.

usage: interp.py <VillagerTrades.javap> <itemmap.json> <blockmap.json>
                 <profmap.json> <Items> <Blocks> <VillagerProfession>
                 <VillagerTrades> [method header]
"""
import re,json,sys
items=json.load(open(sys.argv[2]))
blocks=json.load(open(sys.argv[3]))
prof=json.load(open(sys.argv[4]))
# Inner class letter -> trade type. Check them per version with javap (constructors).
names=json.loads(open(sys.argv[10]).read()) if len(sys.argv)>10 else {'a':'DyedArmor','b':'EmeraldForItems','c':'BoatByType','d':'EnchantBook','e':'EnchantedItem','g':'ItemsAndEmeraldsToItems','h':'ItemsForEmeralds','i':'SuspiciousStew','j':'TippedArrow','k':'TreasureMap'}
itemsCls,blocksCls,profCls,tradesCls=sys.argv[5:9]
lines=open(sys.argv[1]).read().split('\n')
marker=sys.argv[9] if len(sys.argv)>9 else 'static {};'
start=[i for i,l in enumerate(lines) if l.strip()==marker][0]
puts=[]
st=[]
def nargs(desc):
    inner=desc[desc.index('(')+1:desc.index(')')]
    n=0;i=0
    while i<len(inner):
        c=inner[i]
        if c=='L': i=inner.index(';',i)+1; n+=1
        elif c=='[':
            i+=1; continue
        else: i+=1; n+=1
    return n
out=[]
for l in lines[start:]:
    m=re.search(r'\d+: (\w+)\s*(.*)',l)
    if not m: continue
    op,rest=m.groups()
    if op.startswith('iconst_'): st.append(int(op[7:].replace('m','-')))
    elif op in('bipush','sipush'): st.append(int(rest.split()[0]))
    elif op.startswith('fconst_'): st.append(float(op[7:]))
    elif op in ('ldc','ldc_w'):
        v=rest.split('//')[1].strip()
        t,_,val=v.partition(' ')
        st.append(float(val.rstrip('f')) if t=='float' else int(val) if t=='int' else val)
    elif op=='getstatic':
        f=re.search(r'Field (?:([\w$/]+)\.)?(\w+)',rest)
        c,n=f.groups(); c=c or tradesCls
        if c==itemsCls: st.append('I:'+items.get(n,'?'+n))
        elif c==blocksCls: st.append('B:'+blocks.get(n,'?'+n))
        elif c==profCls: st.append('P:'+prof.get(n,'?'+n))
        else: st.append(c+'.'+n)
    elif op=='new': st.append('NEW')
    elif op=='dup': st.append(st[-1])
    elif op in('anewarray','newarray'): st.pop(); st.append([])
    elif op=='aastore':
        v=st.pop(); idx=st.pop(); arr=st.pop()
        if isinstance(arr,list): arr.append(v)
    elif op in('invokespecial','invokestatic','invokevirtual','invokeinterface'):
        f=re.search(r'Method (?:([\w$/]+)\.)?"?([\w<>$]+)"?:(\([^)]*\)\S*)',rest)
        cls,meth,desc=f.groups(); cls=cls or tradesCls
        n=nargs(desc)
        args=[st.pop() for _ in range(n)][::-1] if n else []
        if op!='invokestatic':
            obj=st.pop()
        if meth=='put' and len(args)==2: puts.append(args)
        if meth=='<init>':
            if st and st[-1]=='NEW': st.pop()
            k=cls.split('$')[-1] if cls.startswith(tradesCls+'$') else cls
            st.append({'t':names.get(k,cls),'a':args})
        elif cls=='java/lang/Integer' and meth=='valueOf': st.append(args[0])
        else:
            if desc.endswith(')V'): pass
            else: st.append({'call':cls+'.'+meth,'a':args})
    elif op=='putstatic':
        v=st.pop(); out.append(v)
    elif op=='pop': st and st.pop()
    elif op=='return': break
    elif op=='checkcast': pass
    elif op=='invokedynamic':
        f=re.search(r':(\([^)]*\)\S*)',rest); n=nargs(f.group(1))
        for _ in range(n): st.pop()
        st.append('LAMBDA')
    elif op in('aload_0','astore_0'):
        st.append('?'+op)
json.dump(out if not puts else puts,open('trades_out.json','w'),default=str)
print(len(out))
